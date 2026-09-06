package main

import (
	"fmt"

	"kikundibora/config"
	"kikundibora/database"
)

func main() {
	config.Load()
	database.Connect()
	for _, t := range []string{"users", "members", "contributions", "payment_methods"} {
		var n int64
		database.DB.Raw("SELECT count(*) FROM " + t).Scan(&n)
		fmt.Println(t, n)
	}
	var id string
	database.DB.Raw("SELECT COALESCE(id::text,'-') FROM users WHERE phone='0712345678' AND deleted_at IS NULL").Scan(&id)
	fmt.Println("kkk0001-user:", id)
}
