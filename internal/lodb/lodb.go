package lodb

import(
  "database/sql"
  _ "github.com/mattn/go-sqlite3"
  "hash/adler32"
  "log"
  "strconv"
  "strings"
  "time"
)

type LoQuery struct {
  AuthorID string
  Classes []string
  ChannelID string
  Duration int
  ID uint32 // ID is created by hashing other fields of the LoQuery
  Level int
  Onetime bool
  Timestamp time.Time
  Query string
}

type LoQueryRepository interface {
  Close() error
  Delete(id int) error
  FindByID(id int) (*LoQuery, error)
  FindByAuthorID(authorid string) ([]LoQuery, error)
  Save(query *LoQuery) error
}

type LoRepo struct {
  db *sql.DB
  deleteStmt *sql.Stmt
  findByIDStmt *sql.Stmt
  findByAuthorStmt *sql.Stmt
  saveStmt *sql.Stmt
}

func NewLoRepo(db *sql.DB) *LoRepo {
  deleteStmt, err := db.Prepare(`DELETE FROM lo_queries WHERE id = ?`)
  if err != nil {
    log.Fatal(err)
  }
  findByIDStmt, err := db.Prepare(`SELECT * FROM lo_queries WHERE id = ?`)
  if err != nil {
    log.Fatal(err)
  }
  findByAuthorStmt, err := db.Prepare(`SELECT * FROM lo_queries WHERE author = ?`)
  if err != nil {
    log.Fatal(err)
  }
  saveStmt, err := db.Prepare(`INSERT INTO
    lo_queries (author, classes, channel, duration, level, timestamp, query)
    VALUES (?, ?, ?, ?, ?, ?, ?)`)
  if err != nil {
    log.Fatal(err)
  }

  return &LoRepo{
    db: db,
    deleteStmt: deleteStmt,
    findByIDStmt: findByIDStmt,
    findByAuthorStmt: findByAuthorStmt,
    saveStmt: saveStmt,
  }
}

// Close
func (r *LoRepo) Close() {
  // The DB close is handled in main, so only the
  // prepared statements need to be closed.
  defer r.deleteStmt.Close()
  defer r.findByIDStmt.Close()
  defer r.findByAuthorStmt.Close()
  defer r.saveStmt.Close()
  return
}

// Delete
func (r *LoRepo) Delete(id int) error {
  _, err := r.deleteStmt.Exec(id)
  return err
}

// FindByID
func (r *LoRepo) FindByID(id int) (*LoQuery, error) {
  var q LoQuery
  err := r.findByIDStmt.QueryRow(id).Scan(
    &q.AuthorID,
    &q.Classes,
    &q.ChannelID,
    &q.Duration,
    &q.ID,
    &q.Level,
    &q.Onetime,
    &q.Timestamp,
    &q.Query)
  return &q, err
}

// FindByAuthorID
func (r *LoRepo) FindByAuthor(authorid string) ([]LoQuery, error) {
  var queries []LoQuery

  rows, err := r.findByAuthorStmt.Query(authorid)
  if err != nil {
    return queries, err
  }
  defer rows.Close()

  for rows.Next() {
    var id int
    var q LoQuery
    err := rows.Scan(
      &id,
      &q.AuthorID,
      &q.Classes,
      &q.ChannelID,
      &q.Duration,
      &q.ID,
      &q.Level,
      &q.Onetime,
      &q.Timestamp,
      &q.Query)
    if err != nil {
      log.Fatal(err)
    }

    queries = append(queries, q)
  }
  if err := rows.Err(); err != nil {
    return queries, err
  } else {
    return queries, nil
  }
}

// Save
func (r *LoRepo) Save(q *LoQuery) error {
  // TODO: Handle small chance that there is a hashing conflict
  _, err := r.saveStmt.Exec(
    q.AuthorID,
    ArrToStr(q.Classes),
    q.ChannelID,
    q.Duration,
    q.Hash(),
    q.Level,
    q.Onetime,
    q.Timestamp,
    q.Query)
  return err
}

func ArrToStr(arr []string) string {
  return strings.Join(arr, ", ")
}

func StrToArr(str string) []string {
  return strings.Split(str, ", ")
}

func (q LoQuery) String() string {
  qStrings := []string{}
  qStrings = append(qStrings, "ID: " + strconv.FormatInt(int64(q.ID), 10))
  if q.Query != "" {
    qStrings = append(qStrings, "Query: \"" + q.Query + "\"")
  } else {
    qStrings = append(qStrings, "Query: None")
  }

  if len(q.Classes) > 0 {
    qStrings = append(qStrings, "Classes: " + ArrToStr(q.Classes))
  } else {
    qStrings = append(qStrings, "Classes: None")
  }

  qStrings = append(qStrings, "Timestamp: " + q.Timestamp.Format("Mon Jan _2 15:04:05"))
  qStrings = append(qStrings, "Duration: " + strconv.FormatInt(int64(q.Duration), 10) + "hours")

  if q.Level > 0 {
    qStrings = append(qStrings, "Level: " + strconv.FormatInt(int64(q.Level), 10))
  } else {
    qStrings = append(qStrings, "Level: None")
  }

  qStrings = append(qStrings, "One-time: " + strconv.FormatBool(q.Onetime))

  return strings.Join(qStrings, ", ")
}

func (q *LoQuery) Hash() uint32 {
  b := []byte{}
  b = append(b, []byte(q.Query)...)

  return adler32.Checksum(b)
}
