package repositories

import (
	"errors"
	"strconv"
	"time"

	"backend/internal/models"

	"gorm.io/gorm"
)

type AnalyticsRepository struct {
	db *gorm.DB
}

func NewAnalyticsRepository(db *gorm.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

type BurglaryHotspot struct {
	PoliceStationID int       `json:"police_station_id"`
	Week            time.Time `json:"week"`
	CaseCount       int       `json:"case_count"`
}

func (r *AnalyticsRepository) GetBurglaryHotspots() ([]BurglaryHotspot, error) {
	return r.GetBurglaryHotspotsForUnit(0)
}

func (r *AnalyticsRepository) GetBurglaryHotspotsForUnit(unitID int) ([]BurglaryHotspot, error) {
	var results []BurglaryHotspot
	dialect := r.db.Dialector.Name()

	var rawSQL string
	if dialect == "postgres" {
		rawSQL = `
			SELECT
				cm.PoliceStationID,
				date_trunc('week', cm.CrimeRegisteredDate) as week,
				count(*) as case_count
			FROM CaseMaster cm
			JOIN CrimeSubHead csh ON csh.CrimeSubHeadID = cm.CrimeMinorHeadID
			WHERE csh.CrimeHeadName ILIKE 'Burglary%'
			  AND cm.CrimeRegisteredDate >= current_date - interval '90 days'
			  AND (? = 0 OR cm.PoliceStationID = ?)
			GROUP BY 1, 2
			ORDER BY 2 DESC, 3 DESC;`
	} else {
		rawSQL = `
			SELECT
				cm.PoliceStationID,
				date(cm.CrimeRegisteredDate, 'weekday 0', '-6 days') as week,
				count(*) as case_count
			FROM CaseMaster cm
			JOIN CrimeSubHead csh ON csh.CrimeSubHeadID = cm.CrimeMinorHeadID
			WHERE csh.CrimeHeadName LIKE 'Burglary%'
			  AND cm.CrimeRegisteredDate >= date('now', '-90 days')
			  AND (? = 0 OR cm.PoliceStationID = ?)
			GROUP BY cm.PoliceStationID, week
			ORDER BY week DESC, case_count DESC;`
	}

	rows, err := r.db.Raw(rawSQL, unitID, unitID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var h BurglaryHotspot
		var weekStr string
		if dialect == "postgres" {
			var t time.Time
			if err := rows.Scan(&h.PoliceStationID, &t, &h.CaseCount); err != nil {
				return nil, err
			}
			h.Week = t
		} else {
			if err := rows.Scan(&h.PoliceStationID, &weekStr, &h.CaseCount); err != nil {
				return nil, err
			}
			t, _ := time.Parse("2006-01-02", weekStr)
			h.Week = t
		}
		results = append(results, h)
	}

	return results, nil
}

type RepeatOffender struct {
	AccusedName   string    `json:"accused_name"`
	GenderID      int       `json:"gender_id"`
	DistinctCases int64     `json:"distinct_cases"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
}

func (r *AnalyticsRepository) GetRepeatOffenders(minCases int) ([]RepeatOffender, error) {
	return r.GetRepeatOffendersForUnit(minCases, 0)
}

func (r *AnalyticsRepository) GetRepeatOffendersForUnit(minCases int, unitID int) ([]RepeatOffender, error) {
	var offenders []RepeatOffender
	rawSQL := `
		SELECT
			a.AccusedName,
			a.GenderID,
			count(distinct a.CaseMasterID) as distinct_cases,
			min(cm.CrimeRegisteredDate) as first_seen,
			max(cm.CrimeRegisteredDate) as last_seen
		FROM Accused a
		JOIN CaseMaster cm ON cm.CaseMasterID = a.CaseMasterID
		WHERE (? = 0 OR cm.PoliceStationID = ?)
		GROUP BY a.AccusedName, a.GenderID
		HAVING count(distinct a.CaseMasterID) >= ?
		ORDER BY distinct_cases DESC, last_seen DESC;`

	rows, err := r.db.Raw(rawSQL, unitID, unitID, minCases).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var o RepeatOffender
		var firstStr, lastStr string

		if r.db.Dialector.Name() == "postgres" {
			if err := rows.Scan(&o.AccusedName, &o.GenderID, &o.DistinctCases, &o.FirstSeen, &o.LastSeen); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&o.AccusedName, &o.GenderID, &o.DistinctCases, &firstStr, &lastStr); err != nil {
				return nil, err
			}
			o.FirstSeen, _ = time.Parse("2006-01-02", firstStr[:10])
			o.LastSeen, _ = time.Parse("2006-01-02", lastStr[:10])
		}
		offenders = append(offenders, o)
	}

	return offenders, nil
}

type GraphEdge struct {
	Source      string `json:"source"`
	Target      string `json:"target"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type GraphNode struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

func (r *AnalyticsRepository) GetCoaccusalGraph(accusedID int) ([]GraphNode, []GraphEdge, error) {
	return r.GetCoaccusalGraphForUnit(accusedID, 0)
}

func (r *AnalyticsRepository) GetCoaccusalGraphForUnit(accusedID int, unitID int) ([]GraphNode, []GraphEdge, error) {
	var nodes []GraphNode
	var edges []GraphEdge
	var seedAccused models.Accused

	seedQuery := r.db.Joins("JOIN CaseMaster ON CaseMaster.CaseMasterID = Accused.CaseMasterID")
	if unitID > 0 {
		seedQuery = seedQuery.Where("CaseMaster.PoliceStationID = ?", unitID)
	}
	if err := seedQuery.First(&seedAccused, accusedID).Error; err != nil {
		return nil, nil, err
	}

	nodes = append(nodes, GraphNode{ID: fmtAccusedID(seedAccused.AccusedMasterID), Type: "Person", Label: seedAccused.AccusedName})

	var cases []models.Accused
	casesQuery := r.db.Joins("JOIN CaseMaster ON CaseMaster.CaseMasterID = Accused.CaseMasterID").
		Where("Accused.AccusedName = ? AND Accused.GenderID = ?", seedAccused.AccusedName, seedAccused.GenderID)
	if unitID > 0 {
		casesQuery = casesQuery.Where("CaseMaster.PoliceStationID = ?", unitID)
	}
	if err := casesQuery.Find(&cases).Error; err != nil {
		return nil, nil, err
	}

	caseIDs := make([]int, len(cases))
	for i, c := range cases {
		caseIDs[i] = c.CaseMasterID

		var cm models.CaseMaster
		caseQuery := r.db.Where("CaseMasterID = ?", c.CaseMasterID)
		if unitID > 0 {
			caseQuery = caseQuery.Where("PoliceStationID = ?", unitID)
		}
		if err := caseQuery.First(&cm).Error; err == nil {
			nodes = append(nodes, GraphNode{ID: fmtCaseID(cm.CaseMasterID), Type: "Case", Label: cm.CrimeNo})
			edges = append(edges, GraphEdge{Source: fmtAccusedID(seedAccused.AccusedMasterID), Target: fmtCaseID(cm.CaseMasterID), Type: "ACCUSED_IN", Description: "Accused person in this case"})
		}
	}

	if len(caseIDs) > 0 {
		var coAccused []models.Accused
		coQuery := r.db.Joins("JOIN CaseMaster ON CaseMaster.CaseMasterID = Accused.CaseMasterID").
			Where("Accused.CaseMasterID IN ? AND Accused.AccusedName != ?", caseIDs, seedAccused.AccusedName)
		if unitID > 0 {
			coQuery = coQuery.Where("CaseMaster.PoliceStationID = ?", unitID)
		}
		if err := coQuery.Find(&coAccused).Error; err == nil {
			for _, co := range coAccused {
				nodes = append(nodes, GraphNode{ID: fmtAccusedID(co.AccusedMasterID), Type: "Person", Label: co.AccusedName})
				edges = append(edges, GraphEdge{Source: fmtAccusedID(co.AccusedMasterID), Target: fmtCaseID(co.CaseMasterID), Type: "ACCUSED_IN", Description: "Co-accused in case"})
				edges = append(edges, GraphEdge{Source: fmtAccusedID(seedAccused.AccusedMasterID), Target: fmtAccusedID(co.AccusedMasterID), Type: "CO_ACCUSED_WITH", Description: "Co-accused in one or more cases"})
			}
		}
	}

	nodeMap := make(map[string]GraphNode)
	var uniqueNodes []GraphNode
	for _, n := range nodes {
		if _, ok := nodeMap[n.ID]; !ok {
			nodeMap[n.ID] = n
			uniqueNodes = append(uniqueNodes, n)
		}
	}

	return uniqueNodes, edges, nil
}

func fmtAccusedID(id int) string {
	return "person_" + strconv.Itoa(id)
}

func fmtCaseID(id int) string {
	return "case_" + strconv.Itoa(id)
}

func (r *AnalyticsRepository) GetCaseIntelligenceRecord(caseID, unitID int) (*models.CaseMaster, error) {
	var row models.CaseMaster
	err := r.db.Preload("Complainants").Preload("Victims").Preload("AccusedList").
		Preload("ActsAssociated").Preload("Arrests").Preload("Chargesheet").Preload("OccuranceTime").
		Where("CaseMasterID = ? AND PoliceStationID = ?", caseID, unitID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *AnalyticsRepository) GetCaseDocuments(caseID int) ([]models.CaseDocument, error) {
	var rows []models.CaseDocument
	err := r.db.Where("CaseMasterID = ?", caseID).Find(&rows).Error
	return rows, err
}

func (r *AnalyticsRepository) GetSimilarCaseCandidates(source *models.CaseMaster, unitID int) ([]models.CaseMaster, error) {
	var rows []models.CaseMaster
	from := source.CrimeRegisteredDate.AddDate(-2, 0, 0)
	err := r.db.Preload("ActsAssociated").Preload("CrimeHead").Preload("CrimeSubHead").
		Where("PoliceStationID = ? AND CaseMasterID <> ? AND CrimeRegisteredDate >= ?", unitID, source.CaseMasterID, from).
		Order("CrimeRegisteredDate DESC").Limit(250).Find(&rows).Error
	return rows, err
}

func (r *AnalyticsRepository) GetPendingCaseCandidates(unitID, statusID int) ([]models.CaseMaster, error) {
	var rows []models.CaseMaster
	err := r.db.Preload("Arrests").Preload("Chargesheet").Preload("CrimeHead").Preload("CrimeSubHead").
		Where("PoliceStationID = ? AND CaseStatusID = ?", unitID, statusID).
		Order("CrimeRegisteredDate ASC").Limit(500).Find(&rows).Error
	return rows, err
}
