package nibbler_sql

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	_ "github.com/lib/pq"
	"github.com/markdicksonjr/nibbler"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Extension struct {
	nibbler.Extension

	Models []interface{}
	Db     *gorm.DB

	configuration *Configuration
}

type Configuration struct {
	Scheme   string
	Host     string
	Port     string
	Username string
	Password *string
	Path     string
	Query    url.Values
}

func (s *Extension) Init(app *nibbler.Application) error {
	var err error
	s.configuration, err = s.GetBestConfiguration(app)

	if err != nil {
		return err
	}

	// our postgres driver initializes in way that we can't just plop in the URL
	if s.configuration.Scheme == "postgres" {

		// ensure port is numerical
		if _, err = strconv.Atoi(s.configuration.Port); err != nil {
			return err
		}

		// establish the sslmode from the configuration, defaulting to disable
		sslMode := s.configuration.Query.Get("sslmode")
		if len(sslMode) == 0 {
			sslMode = "disable"
		}

		s.Db, err = gorm.Open(s.configuration.Scheme,
			fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
				s.configuration.Host,
				s.configuration.Port,
				s.configuration.Username,
				s.configuration.Path,
				*s.configuration.Password,
				sslMode,
			))
	} else if s.configuration.Scheme == "sqlite3" {
		s.Db, err = gorm.Open(s.configuration.Scheme, s.configuration.Path)
	} else {
		return errors.New("unknown dialect")
	}

	if err != nil {
		return err
	}

	if s.Db == nil {
		return errors.New("db connection could not be allocated")
	}

	if s.Models != nil {
		for _, v := range s.Models {
			s.Db = s.Db.AutoMigrate(v)
		}
	}

	return nil
}

func (s *Extension) PostInit(app *nibbler.Application) error {
	return nil
}

func (s *Extension) GetName() string {
	return "sql"
}

func (s *Extension) GetConfiguration() *Configuration {
	return s.configuration
}

func (s *Extension) Destroy(app *nibbler.Application) error {
	if s.Db != nil {
		err := s.Db.Close()
		s.Db = nil
		return err
	}
	return nil
}

func IsRecordNotFoundError(err error) bool {
	return gorm.IsRecordNotFoundError(err)
}

func NullifyField(db *gorm.DB, field string) *gorm.DB {
	return db.Update(field, gorm.Expr("NULL"))
}

func (s *Extension) getBestDialect(app *nibbler.Application) (*string, error) {
	var urlParsed *url.URL = nil
	var parseError error = nil

	dbUrl := os.Getenv("DATABASE_URL")

	// parse the url if able
	if len(dbUrl) > 0 {
		urlParsed, parseError = url.Parse(dbUrl)
	}

	// allocate a url if needed
	if urlParsed == nil && len(os.Getenv("DB_DIALECT")) > 0 {
		urlParsed = &url.URL{
			Scheme: os.Getenv("DB_DIALECT"),
		}
	}

	// if we couldn't get a scheme, return the original URL parse error
	if urlParsed == nil {
		return nil, parseError
	}

	// if we ended up getting a scheme, ignore the URL parse error
	return &urlParsed.Scheme, nil
}

// TODO: allow attribute Url on Extension to take precedence over all of this
func (s *Extension) GetBestConfiguration(app *nibbler.Application) (*Configuration, error) {
	var urlParsed *url.URL = nil
	var scheme *string

	// if we got no configuration information from the app, that's an error
	if app.Config == nil {
		return nil, errors.New("sql extension could not get configuration")
	}

	// if the root config is available, attempt to get the SQL URL from it
	if app.Config.Raw != nil {
		dbUrl := app.Config.Raw.Get("sql", "url").String("")

		// parse the url if able, fall back to database.url
		if len(dbUrl) > 0 {
			urlParsed, _ = url.Parse(dbUrl)
		} else {
			dbUrl = app.Config.Raw.Get("database", "url").String("")
		}

		// if we still don't have a URL, parse the url if able, fall back to db.url
		if urlParsed == nil {
			if len(dbUrl) > 0 {
				urlParsed, _ = url.Parse(dbUrl)
			} else {
				dbUrl = app.Config.Raw.Get("db", "url").String("")
			}

			if len(dbUrl) > 0 {
				urlParsed, _ = url.Parse(dbUrl)
			}
		}
	}

	// allocate a url if needed for further operations
	if urlParsed == nil {
		urlParsed = &url.URL{}
	}

	// get the best dialect and apply it as a scheme, if able
	scheme, _ = s.getBestDialect(app)

	if scheme != nil {
		urlParsed.Scheme = *scheme
	}

	// if we couldn't derive a theme, use (default) sqlite3
	if len(urlParsed.Scheme) == 0 {
		schemeVal := "sqlite3"
		urlParsed.Scheme = schemeVal
	}

	// apply fallback user/password
	if urlParsed.User == nil {
		if app.Config.Raw != nil {
			urlParsed.User = url.UserPassword(
				app.Config.Raw.Get("db", "user").String(""),
				app.Config.Raw.Get("db", "password").String(""),
			)
		} else {
			urlParsed.User = url.UserPassword("", "")
		}
	}

	// ensure password is set
	password, isSet := urlParsed.User.Password()

	// parse host/port from url host
	hostParts := strings.Split(urlParsed.Host, ":")

	// if we couldn't get host and port, fall back to other env vars
	if len(hostParts) < 2 {
		newHostParts := make([]string, 2)
		newHostParts[0] = hostParts[0]
		hostParts = newHostParts

		if app.Config.Raw != nil {
			if len(hostParts[0]) == 0 {
				hostParts[0] = app.Config.Raw.Get("db", "host").String("")
			}
			hostParts[1] = app.Config.Raw.Get("db", "port").String("")
		}
	}

	// apply fallback path/name parameter if needed
	if len(urlParsed.Path) == 0 && app.Config.Raw != nil {
		urlParsed.Path = app.Config.Raw.Get("db", "dbname").String("")
	}

	if schemeAcceptsLeadingSlashInPath(urlParsed.Scheme) {
		// the URL parser puts a leading slash on the path, which GORM, etc doesn't like for non-file connections
		urlParsed.Path = strings.Replace(urlParsed.Path, "/", "", -1)
	}

	configuration := &Configuration{
		urlParsed.Scheme,
		hostParts[0],
		hostParts[1],
		urlParsed.User.Username(),
		&password,
		urlParsed.Path,
		urlParsed.Query(),
	}

	// for sqlite, default path to ":memory:"
	if configuration.Scheme == "sqlite3" && len(configuration.Path) == 0 {
		configuration.Path = ":memory:"
	}

	if !isSet {
		configuration.Password = nil
	}

	return configuration, s.validateConfiguration(configuration)
}

func schemeAcceptsLeadingSlashInPath(s string) bool {
	return s != "sqlite3"
}

func (s *Extension) validateConfiguration(configuration *Configuration) error {

	// if we failed to get a configuration, and sqlite3 was not specified
	// we allow sqlite3 to use an in-memory database, so not all of these
	// fields are required for that configuration
	if configuration.Scheme == "sqlite3" {
		return nil
	}

	// ensure host is set
	if len(configuration.Host) == 0 {
		return errors.New("could not find database host parameter in configuration")
	}

	// ensure port is set
	if len(configuration.Port) == 0 {
		return errors.New("could not find database port parameter in configuration")
	}

	// ensure path is set
	if len(configuration.Path) == 0 {
		return errors.New("could not find database path parameter in configuration")
	}

	// ensure user is set
	if len(configuration.Username) == 0 {
		return errors.New("could not find database user parameter in configuration")
	}

	if configuration.Password == nil || len(*configuration.Password) == 0 {
		return errors.New("could not find database password parameter in configuration")
	}

	return nil
}
