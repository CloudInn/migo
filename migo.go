package migo

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/CloudInn/gormigrate/v2"
	"github.com/fatih/color"
	"gorm.io/gorm"
)

type Migrations []*gormigrate.Migration

type gormGooseData struct {
	lastAppliedMigrationID    string
	firstUnAppliedMigrationID string
}

type gormMigration struct {
	ID        int
	VersionID int
	TStamp    time.Time
	IsApplied bool
}

var DefaultOptions *Options = &Options{
	Options: *gormigrate.DefaultOptions,
}

type Options struct {
	gormigrate.Options
	PgSchema string
}

func (o Options) WithPgSchema(pgschema string) *Options {
	d := DefaultOptions
	d.PgSchema = pgschema
	return d
}

var dbClient *gorm.DB
var schemaname     string

var (
	errNoGormGooseMigrationTable error = errors.New("no gorm goose table found")
	errNoAppliedMigrations       error = errors.New("no applied gorm-goose migrations")
	errNoUnAppliedMigrations     error = errors.New("no un-applied gorm-goose migrations")
)

func Run(db *gorm.DB, migrations Migrations, command string, options *Options) error {
	schemaname = options.PgSchema

	if db != nil {
		dbClient = db
	} else {
		log.Fatalln("dbClient is nil")
	}

	options.TableName = fmt.Sprintf("%s.%s", options.PgSchema, options.TableName)
	m := gormigrate.New(db, &options.Options, migrations)

	switch command {
	case "up":
		ggd, err := getGormGooseData()
		if err != nil {
			switch err {
			case errNoGormGooseMigrationTable:
				if err := m.Migrate(); err != nil {
					log.Fatalf("Could not migrate: %v", err)
				}
			case errNoAppliedMigrations:
				if err := m.Migrate(); err != nil {
					log.Fatalf("Could not migrate: %v", err)
				}
			case errNoUnAppliedMigrations:
				if err := m.FakeMigrate(); err != nil {
					log.Fatalf("Could not migrate: %v", err)
				}
			default:
				return err
			}
		} else {
			if err := m.FakeMigrateTo(ggd.lastAppliedMigrationID); err != nil {
				log.Fatalf("Could not migrate: %v", err)
			}
			if err := m.Migrate(); err != nil {
				log.Fatalf("Could not migrate: %v", err)
			}
		}
	case "down":
		if err := m.RollbackLast(); err != nil {
			log.Fatalf("Could not rollback: %v", err)
		}
	default:
		return errors.New("invalid -migrate subcommand (should be neither up, down or gen)")
	}

	color.Green("\n** Migration run successfully **")
	return nil
}

func NewID() string {
	return time.Now().Format("20060102150405")
}


func getGormGooseData() (gormGooseData, error) {
	ggd := gormGooseData{}
	var err error

	if !gormGooseMigrationTableExist() {
		return gormGooseData{}, errNoGormGooseMigrationTable
	}

	ggd.lastAppliedMigrationID, err = getLastGormGooseAppliedMigration()
	if err != nil {
		return gormGooseData{}, err
	}

	ggd.firstUnAppliedMigrationID, err = getFirstGormGooseUnAppliedMigration()
	if err != nil {
		return gormGooseData{}, err
	}
	return ggd, err
}

func gormGooseMigrationTableExist() bool {
	exists := dbClient.Migrator().HasTable(fmt.Sprintf("%s.%s", schemaname, "migration_records"))
	fmt.Println("migration_records exists", exists)
	return exists
}

func getLastGormGooseAppliedMigration() (string, error) {
	gm := gormMigration{}

	result := dbClient.Raw("SELECT * FROM etainvoicing.migration_records WHERE id IN (SELECT MAX(id) FROM etainvoicing.migration_records GROUP BY version_id) AND is_applied = TRUE ORDER BY version_id DESC LIMIT 1;").Scan(&gm)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) || result.RowsAffected == 0 {
		return "", errNoAppliedMigrations
	}
	if result.Error != nil {
		return "", result.Error
	}
	return fmt.Sprint(gm.VersionID), result.Error
}

func getFirstGormGooseUnAppliedMigration() (string, error) {
	gm := gormMigration{}

	result := dbClient.Raw("SELECT * FROM etainvoicing.migration_records WHERE id IN (SELECT MAX(id) FROM etainvoicing.migration_records GROUP BY version_id) AND is_applied = FALSE ORDER BY version_id LIMIT 1;").Scan(&gm)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) || result.RowsAffected == 0 {
		return "", errNoUnAppliedMigrations
	}
	if result.Error != nil {
		return "", result.Error
	}
	return fmt.Sprint(gm.VersionID), result.Error
}
