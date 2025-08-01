package main

import (
	"go-customer-app/database"
	"go-customer-app/handlers"
	"html/template"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func main() {
	// データベースの初期化
	database.InitDB()

	// Ginルーターの初期化
	router := gin.Default()

	// テンプレートで利用するカスタム関数を登録します。
	router.SetFuncMap(template.FuncMap{
		"formatNumber": func(n interface{}) string {
			p := message.NewPrinter(language.Japanese)
			switch v := n.(type) {
			case int:
				return p.Sprintf("%d", v)
			case float64:
				return p.Sprintf("%.2f", v)
			default:
				// 念のため、予期しない型が来ても表示できるようにします。
				return p.Sprintf("%v", n)
			}
		},
		"formatInteger": func(n float64) string {
			p := message.NewPrinter(language.Japanese)
			// float64をintに変換することで小数点以下を切り捨てます。
			return p.Sprintf("%d", int(n))
		},
	})

	// HTMLテンプレートの場所を指定
	router.LoadHTMLGlob("templates/*.html")

	// ルーティング設定
	router.GET("/", handlers.GetCustomers)
	router.GET("/customers/new", handlers.ShowCreateForm)
	router.POST("/customers", handlers.CreateCustomer)
	router.GET("/customers/edit/:id", handlers.ShowEditForm)
	router.POST("/customers/update/:id", handlers.UpdateCustomer)
	router.POST("/customers/delete/:id", handlers.DeleteCustomer)

	// サーバーをポート8080で起動
	router.Run(":8080")
}
