# Migo

Migo is a tool to safely migrate your migrations from gorm-goose to [Gormigrate](https://github.com/go-gormigrate/gormigrate)

### Example

```go
package main

import (
	"gorm.io/driver/sqlite"
	"github.com/CloudInn/migo"
	"gorm.io/gorm"
)


func main() {
	db, _ := gorm.Open(sqlite.Open("gorm.db"), &gorm.Config{})

	migrations := migo.Migrations{
		// create persons table
		{
			ID: "201608301400",
			Migrate: func(tx *gorm.DB) error {
				// it's a good pratice to copy the struct inside the function,
				// so side effects are prevented if the original struct changes during the time
				type Person struct {
					gorm.Model
					Name string
				}
				return tx.AutoMigrate(&Person{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("people")
			},
		},
		// add age column to persons
		{
			ID: "201608301415",
			Migrate: func(tx *gorm.DB) error {
				// when table already exists, it just adds fields as columns
				type Person struct {
					Age int
				}
				return tx.AutoMigrate(&Person{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropColumn("people", "age")
			},
		},
		// add pets table
		{
			ID: "201608301430",
			Migrate: func(tx *gorm.DB) error {
				type Pet struct {
					gorm.Model
					Name     string
					PersonID int
				}
				return tx.AutoMigrate(&Pet{})
			},
			Rollback: func(tx *gorm.DB) error {
				return tx.Migrator().DropTable("pets")
			},
		},
	}

	err := migo.Run(dbClient, migrations, "up", *migo.DefaultOptions.WithPgSchema("<YOUR PGSCHEMA>"))
	if err != nil {
		log.Fatalln(err)
	}

}

```