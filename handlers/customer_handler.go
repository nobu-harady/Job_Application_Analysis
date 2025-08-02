package handlers

import (
	"encoding/json"
	"go-customer-app/database"
	"go-customer-app/models"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ChartDataset はグラフの1本の折れ線に対応するデータセットです。
type ChartDataset struct {
	Label                 string    `json:"label"`
	MonthlyApplications   []int     `json:"monthly_applications"`
	MonthlyRegistrations  []int     `json:"monthly_registrations"`
	MonthlyPlacements     []int     `json:"monthly_placements"`
	ApplicationUnitPrice  []float64 `json:"application_unit_price"`
	RegistrationUnitPrice []float64 `json:"registration_unit_price"`
	PlacementUnitPrice    []float64 `json:"placement_unit_price"`
}

// MultiChartData は複数のデータセットを持つグラフ全体のデータを保持します。
type MultiChartData struct {
	Labels   []string       `json:"labels"`
	Datasets []ChartDataset `json:"datasets"`
}

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
	recruitmentMethod := c.Query("recruitmentMethod")
	customerName := c.Query("customerName")
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")
	sortKey := c.Query("sort")
	order := c.Query("order")

	var customers []models.Customer
	query := database.DB

	// 顧客名が指定されていれば、検索条件に追加 (部分一致)
	if recruitmentMethod != "" {
		// GORMはGoの構造体フィールド名(RecruitmentMethod)をDBのカラム名(recruitment_method)に自動変換するため、カラム名を指定します
		query = query.Where("recruitment_method LIKE ?", "%"+recruitmentMethod+"%")
	}

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

	// ソート順の適用
	if sortKey != "" {
		// orderが "asc" または "desc" であることを確認
		if order != "asc" && order != "desc" {
			order = "asc" // デフォルトは昇順
		}
		// SQLインジェクションを防ぐため、許可されたカラム名かチェック
		allowedSortKeys := map[string]string{
			"year_month":              "year_month",
			"recruitment_method":      "recruitment_method",
			"customer_name":           "customer_name",
			"monthly_fee":             "monthly_fee",
			"monthly_applications":    "monthly_applications",
			"monthly_registrations":   "monthly_registrations",
			"monthly_placements":      "monthly_placements",
			"application_unit_price":  "application_unit_price",
			"registration_unit_price": "registration_unit_price",
			"placement_unit_price":    "placement_unit_price",
			"updated_at":              "updated_at",
		}
		if dbColumn, ok := allowedSortKeys[sortKey]; ok {
			query = query.Order(dbColumn + " " + order)
		} else {
			query = query.Order("year_month desc, id desc")
		}
	} else {
		query = query.Order("year_month desc, id desc")
	}

	query.Find(&customers)

	// --- グラフ用データの準備 ---
	var chartDataJSON template.JS
	if len(customers) > 0 {
		// 1. データを「採用手法 - 顧客名」でグループ化し、同時にすべてのユニークな年月を収集する
		groupedData := make(map[string][]models.Customer)
		uniqueMonths := make(map[time.Time]bool)
		for _, cust := range customers {
			key := cust.RecruitmentMethod + " - " + cust.CustomerName
			groupedData[key] = append(groupedData[key], cust)
			monthStart := time.Date(cust.YearMonth.Year(), cust.YearMonth.Month(), 1, 0, 0, 0, 0, cust.YearMonth.Location())
			uniqueMonths[monthStart] = true
		}

		// 2. ユニークな年月をソートして、グラフのX軸ラベルを作成する
		sortedMonths := make([]time.Time, 0, len(uniqueMonths))
		for month := range uniqueMonths {
			sortedMonths = append(sortedMonths, month)
		}
		sort.Slice(sortedMonths, func(i, j int) bool { return sortedMonths[i].Before(sortedMonths[j]) })

		labels := make([]string, len(sortedMonths))
		monthToIndex := make(map[string]int)
		for i, month := range sortedMonths {
			label := month.Format("2006-01")
			labels[i] = label
			monthToIndex[label] = i
		}

		// 3. グループごとにデータセットを作成する
		chartDatasets := make([]ChartDataset, 0, len(groupedData))
		for label, customerDataList := range groupedData {
			monthlyApplications := make([]int, len(labels))
			monthlyRegistrations := make([]int, len(labels))
			monthlyPlacements := make([]int, len(labels))
			applicationUnitPrice := make([]float64, len(labels))
			registrationUnitPrice := make([]float64, len(labels))
			placementUnitPrice := make([]float64, len(labels))

			for _, cust := range customerDataList {
				if idx, ok := monthToIndex[cust.YearMonth.Format("2006-01")]; ok {
					monthlyApplications[idx] = cust.MonthlyApplications
					monthlyRegistrations[idx] = cust.MonthlyRegistrations
					monthlyPlacements[idx] = cust.MonthlyPlacements
					applicationUnitPrice[idx] = cust.ApplicationUnitPrice
					registrationUnitPrice[idx] = cust.RegistrationUnitPrice
					placementUnitPrice[idx] = cust.PlacementUnitPrice
				}
			}

			dataset := ChartDataset{
				Label:                 label,
				MonthlyApplications:   monthlyApplications,
				MonthlyRegistrations:  monthlyRegistrations,
				MonthlyPlacements:     monthlyPlacements,
				ApplicationUnitPrice:  applicationUnitPrice,
				RegistrationUnitPrice: registrationUnitPrice,
				PlacementUnitPrice:    placementUnitPrice,
			}
			chartDatasets = append(chartDatasets, dataset)
		}

		chartData := MultiChartData{Labels: labels, Datasets: chartDatasets}
		jsonData, _ := json.Marshal(chartData)
		chartDataJSON = template.JS(jsonData)
	}

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
		"customers":         customers,
		"recruitmentMethod": recruitmentMethod,
		"customerName":      customerName,
		"startDate":         startDateStr,
		"endDate":           endDateStr,
		"averages":          averages,
		"sortKey":           sortKey,
		"order":             order,
		"chartData":         chartDataJSON,
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
		RecruitmentMethod:     c.PostForm("recruitmentMethod"),
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
	customer.RecruitmentMethod = c.PostForm("recruitmentMethod")
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
