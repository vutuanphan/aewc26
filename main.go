// Command aewc is the Anh Em WC 2026 friends betting site: a single Go binary
// serving a server-rendered UI backed by SQLite.
package main

import (
	"log"

	_ "time/tzdata" // embed the tz database (Asia/Ho_Chi_Minh) into the binary

	"aewc/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
