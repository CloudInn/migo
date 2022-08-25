package migo

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"
	"flag"

	"github.com/go-gormigrate/gormigrate/v2"
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

var dbClient *gorm.DB

var (
	errNoGormGooseMigrationTable error = errors.New("no gorm goose table found")
	errNoAppliedMigrations       error = errors.New("no applied gorm-goose migrations")
	errNoUnAppliedMigrations     error = errors.New("no un-applied gorm-goose migrations")
)

var (
	schemanameFlag *string
	schemaname     string
)

func init() {
	schemanameFlag = flag.String("pgschema", "", "which postgres-schema to migrate (this will be the pg-schema if SCHEMA_NAME env not set)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "https://github.com/CloudInn/migo\n")
	}
}

func Run(db *gorm.DB, migrations Migrations) error {

	if len(flag.Args()) < 1 {
		fmt.Println("Not enough arguments")
		os.Exit(1)
	}

	// to generate current timestamp to use it as migration id
	if flag.Args()[0] == "gen" {
		fmt.Println(time.Now().Format("20060102150405"))
		os.Exit(0)
		return nil
	}

	if db != nil {
		dbClient = db
	} else {
		log.Fatalln("dbClient is nil")
	}

	schemanameEnv := os.Getenv("SCHEMA_NAME")
	if *schemanameFlag != "" {
		schemaname = *schemanameFlag
	} else if schemanameEnv != "" {
		schemaname = schemanameEnv
	} else {
		return errors.New("no -pgschema flag found or SCHEMA_NAME env")
	}

	migrateOptions := gormigrate.DefaultOptions
	migrateOptions.TableName = fmt.Sprintf("%s.%s", schemaname, gormigrate.DefaultOptions.TableName)
	m := gormigrate.New(db, migrateOptions, migrations)

	switch flag.Args()[0] {
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

	log.Printf("Migration did run")
	return nil
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
