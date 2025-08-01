package handlers

import (
	"go-customer-app/database"
	"go-customer-app/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Averages は表示されているデータの平均値を保持します。
type Averages struct {
	MonthlyFee            float64
	MonthlyApplications   float64
	MonthlyRegistrations  float64
	MonthlyPlacements     float64
	ApplicationUnitPrice  float64
	RegistrationUnitPrice float64
	PlacementUnitPrice    float64
}

// GetCustomers は顧客情報を取得し、一覧ページに表示します。
// クエリパラメータで年月が指定されていれば、そのデータのみをフィルタリングします。
func GetCustomers(c *gin.Context) {
	// クエリパラメータから検索条件を取得
	customerName := c.Query("customerName")
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	var customers []models.Customer
	query := database.DB.Order("year_month desc, id desc")

	// 顧客名が指定されていれば、検索条件に追加 (部分一致)
	if customerName != "" {
		// GORMはGoの構造体フィールド名(CustomerName)をDBのカラム名(customer_name)に自動変換します
		query = query.Where("customer_name LIKE ?", "%"+customerName+"%")
	}

	// 年月が指定されていれば、検索条件に追加
	if startDateStr != "" {
		startDate, err := time.Parse("2006-01", startDateStr)
		if err == nil {
			query = query.Where("year_month >= ?", startDate)
		}
	}
	if endDateStr != "" {
		// 終了日はその月の最終日までを含むように、翌月の初日を基準にします
		endDate, err := time.Parse("2006-01", endDateStr)
		if err == nil {
			query = query.Where("year_month < ?", endDate.AddDate(0, 1, 0))
		}
	}

	query.Find(&customers)

	var averages Averages
	customerCount := len(customers)

	if customerCount > 0 {
		var totalFee, totalApps, totalRegs, totalPlaces int
		var totalAppPrice, totalRegPrice, totalPlacePrice float64

		for _, cust := range customers {
			totalFee += cust.MonthlyFee
			totalApps += cust.MonthlyApplications
			totalRegs += cust.MonthlyRegistrations
			totalPlaces += cust.MonthlyPlacements
			totalAppPrice += cust.ApplicationUnitPrice
			totalRegPrice += cust.RegistrationUnitPrice
			totalPlacePrice += cust.PlacementUnitPrice
		}

		count := float64(customerCount)
		averages = Averages{
			MonthlyFee:            float64(totalFee) / count,
			MonthlyApplications:   float64(totalApps) / count,
			MonthlyRegistrations:  float64(totalRegs) / count,
			MonthlyPlacements:     float64(totalPlaces) / count,
			ApplicationUnitPrice:  totalAppPrice / count,
			RegistrationUnitPrice: totalRegPrice / count,
			PlacementUnitPrice:    totalPlacePrice / count,
		}
	}

	// HTMLをレンダリングし、顧客データを渡す。
	c.HTML(http.StatusOK, "index.html", gin.H{
		"customers":    customers,
		"customerName": customerName,
		"startDate":    startDateStr,
		"endDate":      endDateStr,
		"averages":     averages,
	})
}

// ShowCreateForm は新規顧客登録ページを表示します。
// ShowCreateForm は新規顧客登録ページを表示します。
func ShowCreateForm(c *gin.Context) {
	now := time.Now()
	// 登録フォームに現在の年月をデフォルト値として渡します。
	c.HTML(http.StatusOK, "create.html", gin.H{
		"currentYearMonth": now.Format("2006-01"),
	})
}

// calculateUnitPrice は費用と件数から単価を計算するヘルパー関数です。
// 件数が0の場合は0を返します。
func calculateUnitPrice(fee, count int) float64 {
	if count > 0 {
		return float64(fee) / float64(count)
	}
	return 0.0
}

// CreateCustomer はフォームから送信されたデータで新しい顧客を作成します。
func CreateCustomer(c *gin.Context) {
	// フォームデータをパースする。
	customerName := c.PostForm("customerName")

	yearMonthStr := c.PostForm("yearMonth")
	yearMonth, err := time.Parse("2006-01", yearMonthStr)
	if err != nil {
		c.String(http.StatusBadRequest, "年月の値が無効です: %v", err)
		return
	}

	monthlyFee, err := strconv.Atoi(c.PostForm("monthlyFee"))
	if err != nil {
		c.String(http.StatusBadRequest, "月額費用の値が無効です: %v", err)
		return
	}
	monthlyApplications, err := strconv.Atoi(c.PostForm("monthlyApplications"))
	if err != nil {
		c.String(http.StatusBadRequest, "月の応募数の値が無効です: %v", err)
		return
	}
	monthlyRegistrations, err := strconv.Atoi(c.PostForm("monthlyRegistrations"))
	if err != nil {
		c.String(http.StatusBadRequest, "月の登録数の値が無効です: %v", err)
		return
	}
	monthlyPlacements, err := strconv.Atoi(c.PostForm("monthlyPlacements"))
	if err != nil {
		c.String(http.StatusBadRequest, "月の就業数の値が無効です: %v", err)
		return
	}

	// 単価を計算します。分母が0の場合は0とします。
	placementUnitPrice := calculateUnitPrice(monthlyFee, monthlyPlacements)
	registrationUnitPrice := calculateUnitPrice(monthlyFee, monthlyRegistrations)
	applicationUnitPrice := calculateUnitPrice(monthlyFee, monthlyApplications)

	// 新しいCustomerインスタンスを作成します。
	customer := models.Customer{
		CustomerName:          customerName,
		YearMonth:             yearMonth,
		MonthlyFee:            monthlyFee,
		MonthlyApplications:   monthlyApplications,
		MonthlyRegistrations:  monthlyRegistrations,
		MonthlyPlacements:     monthlyPlacements,
		PlacementUnitPrice:    placementUnitPrice,
		RegistrationUnitPrice: registrationUnitPrice,
		ApplicationUnitPrice:  applicationUnitPrice,
	}

	// DBに保存します。
	result := database.DB.Create(&customer)
	if result.Error != nil {
		// エラーハンドリング（例: エラーページを表示）
		c.String(http.StatusInternalServerError, "データの作成に失敗しました: %v", result.Error)
		return
	}

	// 成功したらルートページにリダイレクトします。
	c.Redirect(http.StatusFound, "/")
}

// ShowEditForm は編集ページを表示します。
func ShowEditForm(c *gin.Context) {
	id := c.Param("id")
	var customer models.Customer
	if err := database.DB.First(&customer, id).Error; err != nil {
		c.String(http.StatusNotFound, "指定された顧客が見つかりません: %v", err)
		return
	}

	c.HTML(http.StatusOK, "edit.html", gin.H{
		"customer": customer,
	})
}

// UpdateCustomer は顧客情報を更新します。
func UpdateCustomer(c *gin.Context) {
	id := c.Param("id")
	var customer models.Customer
	if err := database.DB.First(&customer, id).Error; err != nil {
		c.String(http.StatusNotFound, "指定された顧客が見つかりません: %v", err)
		return
	}

	// フォームデータをパースします。
	customer.CustomerName = c.PostForm("customerName")
	yearMonthStr := c.PostForm("yearMonth")
	yearMonth, err := time.Parse("2006-01", yearMonthStr)
	if err != nil {
		c.String(http.StatusBadRequest, "年月の値が無効です: %v", err)
		return
	}
	customer.YearMonth = yearMonth

	customer.MonthlyFee, err = strconv.Atoi(c.PostForm("monthlyFee"))
	if err != nil {
		c.String(http.StatusBadRequest, "月額費用の値が無効です: %v", err)
		return
	}
	customer.MonthlyApplications, err = strconv.Atoi(c.PostForm("monthlyApplications"))
	if err != nil {
		c.String(http.StatusBadRequest, "月の応募数の値が無効です: %v", err)
		return
	}
	customer.MonthlyRegistrations, err = strconv.Atoi(c.PostForm("monthlyRegistrations"))
	if err != nil {
		c.String(http.StatusBadRequest, "月の登録数の値が無効です: %v", err)
		return
	}
	customer.MonthlyPlacements, err = strconv.Atoi(c.PostForm("monthlyPlacements"))
	if err != nil {
		c.String(http.StatusBadRequest, "月の就業数の値が無効です: %v", err)
		return
	}

	// 単価を再計算します。
	customer.PlacementUnitPrice = calculateUnitPrice(customer.MonthlyFee, customer.MonthlyPlacements)
	customer.RegistrationUnitPrice = calculateUnitPrice(customer.MonthlyFee, customer.MonthlyRegistrations)
	customer.ApplicationUnitPrice = calculateUnitPrice(customer.MonthlyFee, customer.MonthlyApplications)

	// データベースに保存します。
	if err := database.DB.Save(&customer).Error; err != nil {
		c.String(http.StatusInternalServerError, "データの更新に失敗しました: %v", err)
		return
	}

	c.Redirect(http.StatusFound, "/")
}

// DeleteCustomer は顧客情報を削除します。
func DeleteCustomer(c *gin.Context) {
	id := c.Param("id")
	// GORMのDeleteメソッドはソフトデリート（deleted_atにタイムスタンプを設定）を実行します。
	if err := database.DB.Delete(&models.Customer{}, id).Error; err != nil {
		c.String(http.StatusInternalServerError, "データの削除に失敗しました: %v", err)
		return
	}
	c.Redirect(http.StatusFound, "/")
}
