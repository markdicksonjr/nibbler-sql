package main

import (
	"github.com/markdicksonjr/nibbler"
	sql "github.com/markdicksonjr/nibbler-sql"
	"github.com/markdicksonjr/nibbler-sql/session"
	NibUserSql "github.com/markdicksonjr/nibbler-sql/user"
	"github.com/markdicksonjr/nibbler/session"
	"github.com/markdicksonjr/nibbler/user"
	_ "github.com/michaeljs1990/sqlitestore"
	"log"
)

func main() {

	// allocate configuration
	config, err := nibbler.LoadConfiguration()

	if err != nil {
		log.Fatal(err)
	}

	// prepare models for initialization
	var models = []interface{}{
		nibbler.User{},
	}

	// allocate an SQL controller, providing an sql extension
	sqlController := NibUserSql.Extension{
		SqlExtension: &sql.Extension{
			Models: models,
		},
	}

	// allocate user extension, providing sql extension to it
	userExtension := user.Extension{
		PersistenceExtension: &sqlController, // &elasticController,
	}

	// allocate session extension, with an optional custom connector
	var sessionConnector session.StoreConnector = &connectors.SqlStoreConnector{
		Secret:       "somesecret",
		SqlExtension: sqlController.SqlExtension,
	}
	sessionExtension := session.Extension{
		StoreConnector: sessionConnector,
	}

	// initialize the application, provide config, logger, extensions
	appContext := nibbler.Application{}
	if err = appContext.Init(config, nibbler.DefaultLogger{}, []nibbler.Extension{
		sqlController.SqlExtension,
		&userExtension,
		&sessionExtension,
	}); err != nil {
		log.Fatal(err.Error())
	}

	// start the app
	if err = appContext.Run(); err != nil {
		log.Fatal(err.Error())
	}
}
