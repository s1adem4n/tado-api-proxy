package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_2442875294")
		if err != nil {
			return err
		}

		// add field
		if err := collection.Fields.AddMarshaledJSONAt(6, []byte(`{
			"hidden": false,
			"id": "select961728715",
			"maxSelect": 1,
			"name": "platform",
			"presentable": false,
			"required": true,
			"system": false,
			"type": "select",
			"values": [
				"web",
				"mobile"
			]
		}`)); err != nil {
			return err
		}

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_2442875294")
		if err != nil {
			return err
		}

		// remove field
		collection.Fields.RemoveById("select961728715")

		return app.Save(collection)
	})
}
