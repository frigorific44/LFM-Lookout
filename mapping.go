package main

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/simple"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/mapping"
)

func buildIndexMapping() (mapping.IndexMapping, error) {
	// a generic reusable mapping for english text
	englishTextFieldMapping := bleve.NewTextFieldMapping()
	englishTextFieldMapping.Analyzer = en.AnalyzerName

	// a generic reusable mapping for keyword text
	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Analyzer = keyword.Name

	// a generic reusable mapping for simple text
	simpleFieldMapping := bleve.NewTextFieldMapping()
	simpleFieldMapping.Analyzer = simple.Name

	// a generic reusable mapping for booleans
	booleanFieldMapping := bleve.NewBooleanFieldMapping()

	// a generic reusable mapping for numbers
	numericFieldMapping := bleve.NewNumericFieldMapping()

	//SearchableGroup
	sGroupMapping := bleve.NewDocumentMapping()
	//  Server
	sGroupMapping.AddFieldMappingsAt("Server", keywordFieldMapping)
	//  Members
	sGroupMapping.AddFieldMappingsAt("Members", numericFieldMapping)
	//  Fresh
	sGroupMapping.AddFieldMappingsAt("Fresh", booleanFieldMapping)
	//  Group
	groupMapping := bleve.NewDocumentMapping()
	//    Id
	groupMapping.AddFieldMappingsAt("Id", keywordFieldMapping)
	//    Comment
	groupMapping.AddFieldMappingsAt("Comment", englishTextFieldMapping)
	//    Difficulty
	groupMapping.AddFieldMappingsAt("Difficulty", simpleFieldMapping)
	//    MinLevel
	groupMapping.AddFieldMappingsAt("MinimumLevel", numericFieldMapping)
	//    MaxLevel
	groupMapping.AddFieldMappingsAt("MaximumLevel", numericFieldMapping)
	//    AdventureActive
	groupMapping.AddFieldMappingsAt("AdventureActive", numericFieldMapping)
	//    Leader
	//      Location
	//        Name
	//        Region
	//    Members
	//      Location
	//        Name
	//        Region
	memberMapping := bleve.NewDocumentMapping()
	locationMapping := bleve.NewDocumentMapping()
	locationMapping.AddFieldMappingsAt("Name", englishTextFieldMapping)
	locationMapping.AddFieldMappingsAt("Region", englishTextFieldMapping)
	memberMapping.AddSubDocumentMapping("Location", locationMapping)

	groupMapping.AddSubDocumentMapping("Leader", memberMapping)
	groupMapping.AddSubDocumentMapping("Members", memberMapping)

	//    Quest
	questMapping := bleve.NewDocumentMapping()
	//      Name
	questMapping.AddFieldMappingsAt("Name", englishTextFieldMapping)
	//      AdventurePack
	questMapping.AddFieldMappingsAt("RequiredAdventurePack", englishTextFieldMapping)
	//      AdventureArea
	questMapping.AddFieldMappingsAt("AdventureArea", englishTextFieldMapping)
	//      QuestJournalGroup
	questMapping.AddFieldMappingsAt("QuestJournalGroup", englishTextFieldMapping)
	//      GroupSize
	questMapping.AddFieldMappingsAt("GroupSize", simpleFieldMapping)
	//      Patron
	questMapping.AddFieldMappingsAt("Patron", englishTextFieldMapping)

	groupMapping.AddSubDocumentMapping("Quest", questMapping)

	sGroupMapping.AddSubDocumentMapping("Group", groupMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping("SearchableGroup", sGroupMapping)

	indexMapping.TypeField = "type"
	indexMapping.DefaultAnalyzer = "en"
	indexMapping.DefaultMapping = sGroupMapping

	return indexMapping, nil
}
