package lodb

import(
  "errors"
  "fmt"
  "strings"
  "time"
  "unicode/utf8"

  badger "github.com/dgraph-io/badger/v3"
)

var (
  ErrOrphanPair = errors.New("Key-value entry has no corresponding query or return entry.")
  ErrUserIndicesFull = errors.New("No open index found for user.")
  ErrMalformedKey = errors.New("The key appears malformed and cannot be processed.")
  ErrCorruptQuery = errors.New("A part of the query is corrupted.")
)

const (
  // The minimum value a query's user-unique id can be.
  IDMIN rune = 0
  // The maximum value a query's user-uniqe id can be.
  IDMAX rune = 9
  // The maximum life-time of a query.
  TTLMAX time.Duration = time.Hour * 24
  TICKPERIOD rune = 24 * 60 * 2
)

type LoQuery struct {
  AuthorID string
  ChannelID string
  ID rune // ID is a unique-per-user integer, 0-9.
  TTL time.Duration
  Query string // String formatted for string query in Bleve
}


type LoRepo struct {
  db *badger.DB
}

func NewLoRepo(path string) (*LoRepo, error) {
  db, err := badger.Open(badger.DefaultOptions(path))
  if err != nil {
    return &LoRepo{db: db}, err
  }
  return &LoRepo{db: db}, nil
}

// Close
func (r *LoRepo) Close() {
  r.db.Close()
}

// Delete
func (r *LoRepo) Delete(authorID string, id rune) error {
  err := r.db.Update(func(txn *badger.Txn) error {
    qKey := fmt.Sprintf("query-%s-%s", authorID, string(id))
    rKey := fmt.Sprintf("return-%s-%s", authorID, string(id))

    err1 := txn.Delete([]byte(qKey))
    if err1 != nil {
      return err1
    }
    err2 := txn.Delete([]byte(rKey))
    if err2 != nil {
      return err2
    }
    return nil
  })
  if err != nil {
    return err
  }
  return nil
}


// FindByAuthorID
func (r *LoRepo) FindByAuthor(authorid string) ([]LoQuery, error) {
  var queries []LoQuery

  err := r.db.View(func(txn *badger.Txn) error {
    it := txn.NewIterator(badger.DefaultIteratorOptions)
    defer it.Close()
    prefix := []byte(fmt.Sprintf("query-%s", authorid))
    for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
      qItem := it.Item()
      var valCopy []byte
      keyStr := string(qItem.Key())
      err := qItem.Value(func(v []byte) error {
        valCopy = append([]byte{}, v...)
        return nil
      })
      if err != nil {
        return err
      }
      if len(keyStr) <= len("query-") + len("-0") {
          return ErrMalformedKey
      }
      // Parse out the AuthorID.
      auth := keyStr[len("query-"):len(keyStr)-len("-0")]

      r, _ := utf8.DecodeLastRuneInString(keyStr)
      id, _ := DecodeFinalRune(r)
      if id < IDMIN || id > IDMAX {
        return ErrMalformedKey
      }
      // Retrieve the current TTL.
      expTime := time.Unix(int64(qItem.ExpiresAt()), 0)
      ttl := expTime.Sub(time.Now())
      // TODO: check for valid ttl (not max or min, is less than 24 hours, ect.)
      if ttl > TTLMAX {
        return ErrCorruptQuery
      }

      loQ := LoQuery{
        AuthorID: auth,
        ChannelID: "",
        ID: r,
        TTL: ttl,
        Query: string(valCopy),
      }
      queries = append(queries, loQ)
    }
    return nil
  })
  if err != nil {
    return queries, err
  }
  return queries, nil
}

// Save // TODO: Change to return ID?
func (r *LoRepo) Save(q LoQuery, tick rune) error {
  // Find lowest unused index.
  err := r.db.Update(func(txn *badger.Txn) error {
    itOpts := badger.DefaultIteratorOptions
    itOpts.PrefetchValues = false
    it := txn.NewIterator(itOpts)
    defer it.Close()
    prefix := []byte("query-"+q.AuthorID)
    unusedIndex := IDMIN
    for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
      item := it.Item()
      k := item.Key()
      r, _ := utf8.DecodeLastRuneInString(string(k))
      curr, _ := DecodeFinalRune(r)
      if unusedIndex < curr {
        break
      } else {
        unusedIndex = curr + 1
      }
    }
    if unusedIndex > IDMAX {
      return ErrUserIndicesFull
    }
    // Set unused index to new query.
    // query-[AuthorID]-[君-贤][0-9]:[Query]
    fr := EncodeFinalRune(unusedIndex, tick)
    qKey := fmt.Sprintf(`query-%s-%s`, q.AuthorID, string(fr))
    e1 := badger.NewEntry([]byte(qKey), []byte(q.Query)).WithTTL(q.TTL)
    _ = txn.SetEntry(e1)
    // return-[AuthorID]-[君-贤][0-9]:[ChannelID]
    rKey := fmt.Sprintf(`return-%s-%s`, q.AuthorID, string(fr))
    e2 := badger.NewEntry([]byte(rKey), []byte(q.ChannelID)).WithTTL(q.TTL)
    _ = txn.SetEntry(e2)

    return nil
  })
  if err != nil {
    return err
  }
  return nil
}

func (r *LoRepo) GetView(fn func(txn *badger.Txn) error) error {
  return r.db.View(fn)
}

func (q LoQuery) String() string {
  qStrings := []string{}
  qStrings = append(qStrings, fmt.Sprintf("ID: %X", q.ID))
  qStrings = append(qStrings, "Query: \"" + q.Query + "\"")

  qStrings = append(qStrings, "Duration: " + q.TTL.String())

  return strings.Join(qStrings, ", ")
}

func EncodeFinalRune(id rune, tick rune) rune {
  return (id * TICKPERIOD) + tick
}

func DecodeFinalRune(r rune) (rune, rune) {
  id := r / TICKPERIOD
  tick := r - (id * TICKPERIOD)
  return id, tick
}

func NextTickRune(r rune) rune {
  return (r + 1) % TICKPERIOD
}

func GetIDFromKey(k string) rune {
  r, _ := utf8.DecodeLastRuneInString(k)
  return r
}
