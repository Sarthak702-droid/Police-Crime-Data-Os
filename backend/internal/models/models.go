package models

import (
	"time"
)

// --- ORGANIZATION MASTERS ---

type State struct {
	StateID      int    `gorm:"primaryKey;column:StateID" json:"state_id"`
	StateName    string `gorm:"column:StateName;type:varchar(100)" json:"state_name"`
	NationalityID int    `gorm:"column:NationalityID" json:"nationality_id"`
	Active       bool   `gorm:"column:Active;type:bit" json:"active"`
}

func (State) TableName() string { return "State" }

type District struct {
	DistrictID   int    `gorm:"primaryKey;column:DistrictID" json:"district_id"`
	DistrictName string `gorm:"column:DistrictName;type:varchar(100)" json:"district_name"`
	StateID      int    `gorm:"column:StateID" json:"state_id"`
	Active       bool   `gorm:"column:Active;type:bit" json:"active"`
	State        State  `gorm:"foreignKey:StateID;references:StateID" json:"state,omitempty"`
}

func (District) TableName() string { return "District" }

type UnitType struct {
	UnitTypeID   int    `gorm:"primaryKey;column:UnitTypeID" json:"unit_type_id"`
	UnitTypeName string `gorm:"column:UnitTypeName;type:varchar(100)" json:"unit_type_name"`
	CityDistState string `gorm:"column:CityDistState;type:varchar(50)" json:"city_dist_state"`
	Hierarchy    int    `gorm:"column:Hierarchy" json:"hierarchy"`
	Active       bool   `gorm:"column:Active;type:bit" json:"active"`
}

func (UnitType) TableName() string { return "UnitType" }

type Unit struct {
	UnitID        int      `gorm:"primaryKey;column:UnitID" json:"unit_id"`
	UnitName      string   `gorm:"column:UnitName;type:varchar(100)" json:"unit_name"`
	TypeID        int      `gorm:"column:TypeID" json:"type_id"`
	ParentUnit    *int     `gorm:"column:ParentUnit" json:"parent_unit,omitempty"`
	NationalityID int      `gorm:"column:NationalityID" json:"nationality_id"`
	StateID       int      `gorm:"column:StateID" json:"state_id"`
	DistrictID    int      `gorm:"column:DistrictID" json:"district_id"`
	Active        bool     `gorm:"column:Active;type:bit" json:"active"`
	UnitType      UnitType `gorm:"foreignKey:TypeID;references:UnitTypeID" json:"unit_type,omitempty"`
	District      District `gorm:"foreignKey:DistrictID;references:DistrictID" json:"district,omitempty"`
	State         State    `gorm:"foreignKey:StateID;references:StateID" json:"state,omitempty"`
}

func (Unit) TableName() string { return "Unit" }

type Rank struct {
	RankID    int    `gorm:"primaryKey;column:RankID" json:"rank_id"`
	RankName  string `gorm:"column:RankName;type:varchar(100)" json:"rank_name"`
	Hierarchy int    `gorm:"column:Hierarchy" json:"hierarchy"`
	Active    bool   `gorm:"column:Active;type:bit" json:"active"`
}

func (Rank) TableName() string { return "Rank" }

type Designation struct {
	DesignationID   int    `gorm:"primaryKey;column:DesignationID" json:"designation_id"`
	DesignationName string `gorm:"column:DesignationName;type:varchar(100)" json:"designation_name"`
	Active          bool   `gorm:"column:Active;type:bit" json:"active"`
	SortOrder       int    `gorm:"column:SortOrder" json:"sort_order"`
}

func (Designation) TableName() string { return "Designation" }

type Employee struct {
	EmployeeID          int         `gorm:"primaryKey;column:EmployeeID" json:"employee_id"`
	DistrictID          int         `gorm:"column:DistrictID" json:"district_id"`
	UnitID              int         `gorm:"column:UnitID" json:"unit_id"`
	RankID              int         `gorm:"column:RankID" json:"rank_id"`
	DesignationID       int         `gorm:"column:DesignationID" json:"designation_id"`
	KGID                string      `gorm:"column:KGID;type:varchar(50);uniqueIndex" json:"kgid"`
	FirstName           string      `gorm:"column:FirstName;type:varchar(100)" json:"first_name"`
	EmployeeDOB         time.Time   `gorm:"column:EmployeeDOB;type:date" json:"employee_dob"`
	GenderID            int         `gorm:"column:GenderID" json:"gender_id"`
	BloodGroupID        int         `gorm:"column:BloodGroupID" json:"blood_group_id"`
	PhysicallyChallenged bool       `gorm:"column:PhysicallyChallenged;type:bit" json:"physically_challenged"`
	AppointmentDate     time.Time   `gorm:"column:AppointmentDate;type:date" json:"appointment_date"`
	District            District    `gorm:"foreignKey:DistrictID;references:DistrictID" json:"district,omitempty"`
	Unit                Unit        `gorm:"foreignKey:UnitID;references:UnitID" json:"unit,omitempty"`
	Rank                Rank        `gorm:"foreignKey:RankID;references:RankID" json:"rank,omitempty"`
	Designation         Designation `gorm:"foreignKey:DesignationID;references:DesignationID" json:"designation,omitempty"`
}

func (Employee) TableName() string { return "Employee" }

// --- CRIME CLASSIFICATION & LEGAL MASTERS ---

type CaseCategory struct {
	CaseCategoryID int    `gorm:"primaryKey;column:CaseCategoryID" json:"case_category_id"`
	LookupValue    string `gorm:"column:LookupValue;type:varchar(50)" json:"lookup_value"` // FIR, UDR, PAR...
}

func (CaseCategory) TableName() string { return "CaseCategory" }

type GravityOffence struct {
	GravityOffenceID int    `gorm:"primaryKey;column:GravityOffenceID" json:"gravity_offence_id"`
	LookupValue      string `gorm:"column:LookupValue;type:varchar(100)" json:"lookup_value"` // Heinous, Non-Heinous...
}

func (GravityOffence) TableName() string { return "GravityOffence" }

type CrimeHead struct {
	CrimeHeadID    int    `gorm:"primaryKey;column:CrimeHeadID" json:"crime_head_id"`
	CrimeGroupName string `gorm:"column:CrimeGroupName;type:varchar(100)" json:"crime_group_name"`
	Active         bool   `gorm:"column:Active;type:bit" json:"active"`
}

func (CrimeHead) TableName() string { return "CrimeHead" }

type CrimeSubHead struct {
	CrimeSubHeadID int       `gorm:"primaryKey;column:CrimeSubHeadID" json:"crime_sub_head_id"`
	CrimeHeadID    int       `gorm:"column:CrimeHeadID" json:"crime_head_id"`
	CrimeHeadName  string    `gorm:"column:CrimeHeadName;type:varchar(100)" json:"crime_head_name"`
	SeqID          int       `gorm:"column:SeqID" json:"seq_id"`
	CrimeHead      CrimeHead `gorm:"foreignKey:CrimeHeadID;references:CrimeHeadID" json:"crime_head,omitempty"`
}

func (CrimeSubHead) TableName() string { return "CrimeSubHead" }

type Act struct {
	ActCode        string `gorm:"primaryKey;column:ActCode;type:varchar(50)" json:"act_code"`
	ActDescription string `gorm:"column:ActDescription;type:varchar(255)" json:"act_description"`
	ShortName      string `gorm:"column:ShortName;type:varchar(50)" json:"short_name"`
	Active         bool   `gorm:"column:Active;type:bit" json:"active"`
}

func (Act) TableName() string { return "Act" }

type Section struct {
	ActCode            string `gorm:"primaryKey;column:ActCode;type:varchar(50)" json:"act_code"`
	SectionCode        string `gorm:"primaryKey;column:SectionCode;type:varchar(50)" json:"section_code"`
	SectionDescription string `gorm:"column:SectionDescription;type:text" json:"section_description"`
	Active             bool   `gorm:"column:Active;type:bit" json:"active"`
	Act                Act    `gorm:"foreignKey:ActCode;references:ActCode" json:"act,omitempty"`
}

func (Section) TableName() string { return "Section" }

type CrimeHeadActSection struct {
	CrimeHeadID int     `gorm:"primaryKey;column:CrimeHeadID" json:"crime_head_id"`
	ActCode     string  `gorm:"primaryKey;column:ActCode;type:varchar(50)" json:"act_code"`
	SectionCode string  `gorm:"primaryKey;column:SectionCode;type:varchar(50)" json:"section_code"`
	CrimeHead   CrimeHead `gorm:"foreignKey:CrimeHeadID;references:CrimeHeadID" json:"crime_head,omitempty"`
	Act         Act     `gorm:"foreignKey:ActCode;references:ActCode" json:"act,omitempty"`
}

func (CrimeHeadActSection) TableName() string { return "CrimeHeadActSection" }

// --- DEMOGRAPHIC MASTERS ---

type CasteMaster struct {
	CasteMasterID   int    `gorm:"primaryKey;column:caste_master_id" json:"caste_master_id"`
	CasteMasterName string `gorm:"column:caste_master_name;type:varchar(100)" json:"caste_master_name"`
}

func (CasteMaster) TableName() string { return "CasteMaster" }

type ReligionMaster struct {
	ReligionID   int    `gorm:"primaryKey;column:ReligionID" json:"religion_id"`
	ReligionName string `gorm:"column:ReligionName;type:varchar(100)" json:"religion_name"`
}

func (ReligionMaster) TableName() string { return "ReligionMaster" }

type OccupationMaster struct {
	OccupationID   int    `gorm:"primaryKey;column:OccupationID" json:"occupation_id"`
	OccupationName string `gorm:"column:OccupationName;type:varchar(100)" json:"occupation_name"`
}

func (OccupationMaster) TableName() string { return "OccupationMaster" }

type CaseStatusMaster struct {
	CaseStatusID   int    `gorm:"primaryKey;column:CaseStatusID" json:"case_status_id"`
	CaseStatusName string `gorm:"column:CaseStatusName;type:varchar(100)" json:"case_status_name"`
}

func (CaseStatusMaster) TableName() string { return "CaseStatusMaster" }

type Court struct {
	CourtID    int      `gorm:"primaryKey;column:CourtID" json:"court_id"`
	CourtName  string   `gorm:"column:CourtName;type:varchar(100)" json:"court_name"`
	DistrictID int      `gorm:"column:DistrictID" json:"district_id"`
	StateID    int      `gorm:"column:StateID" json:"state_id"`
	Active     bool     `gorm:"column:Active;type:bit" json:"active"`
	District   District `gorm:"foreignKey:DistrictID;references:DistrictID" json:"district,omitempty"`
	State      State    `gorm:"foreignKey:StateID;references:StateID" json:"state,omitempty"`
}

func (Court) TableName() string { return "Court" }

// --- CORE TRANSACTIONAL CORE TABLES ---

type CaseMaster struct {
	CaseMasterID        int              `gorm:"primaryKey;autoIncrement;column:CaseMasterID" json:"case_master_id"`
	CrimeNo             string           `gorm:"column:CrimeNo;type:varchar(50);uniqueIndex" json:"crime_no"`
	CaseNo              string           `gorm:"column:CaseNo;type:varchar(50)" json:"case_no"`
	CrimeRegisteredDate time.Time        `gorm:"column:CrimeRegisteredDate;type:date" json:"crime_registered_date"`
	PolicePersonID      int              `gorm:"column:PolicePersonID" json:"police_person_id"`
	PoliceStationID     int              `gorm:"column:PoliceStationID;index" json:"police_station_id"`
	CaseCategoryID      int              `gorm:"column:CaseCategoryID" json:"case_category_id"`
	GravityOffenceID    int              `gorm:"column:GravityOffenceID" json:"gravity_offence_id"`
	CrimeMajorHeadID    int              `gorm:"column:CrimeMajorHeadID" json:"crime_major_head_id"`
	CrimeMinorHeadID    int              `gorm:"column:CrimeMinorHeadID" json:"crime_minor_head_id"`
	CaseStatusID        int              `gorm:"column:CaseStatusID" json:"case_status_id"`
	CourtID             int              `gorm:"column:CourtID" json:"court_id"`
	IncidentFromDate    time.Time        `gorm:"column:IncidentFromDate;type:datetime" json:"incident_from_date"`
	IncidentToDate      time.Time        `gorm:"column:IncidentToDate;type:datetime" json:"incident_to_date"`
	InfoReceivedPSDate  time.Time        `gorm:"column:InfoReceivedPSDate;type:datetime" json:"info_received_ps_date"`
	Latitude            float64          `gorm:"column:latitude;type:decimal(9,6)" json:"latitude"`
	Longitude           float64          `gorm:"column:longitude;type:decimal(9,6)" json:"longitude"`
	BriefFacts          string           `gorm:"column:BriefFacts;type:text" json:"brief_facts"`
	PolicePerson        *Employee        `gorm:"foreignKey:PolicePersonID;references:EmployeeID" json:"police_person,omitempty"`
	PoliceStation       *Unit            `gorm:"foreignKey:PoliceStationID;references:UnitID" json:"police_station,omitempty"`
	CaseCategory        *CaseCategory    `gorm:"foreignKey:CaseCategoryID;references:CaseCategoryID" json:"case_category,omitempty"`
	GravityOffence      *GravityOffence  `gorm:"foreignKey:GravityOffenceID;references:GravityOffenceID" json:"gravity_offence,omitempty"`
	CrimeHead           *CrimeHead       `gorm:"foreignKey:CrimeMajorHeadID;references:CrimeHeadID" json:"crime_head,omitempty"`
	CrimeSubHead        *CrimeSubHead    `gorm:"foreignKey:CrimeMinorHeadID;references:CrimeSubHeadID" json:"crime_sub_head,omitempty"`
	CaseStatus          *CaseStatusMaster `gorm:"foreignKey:CaseStatusID;references:CaseStatusID" json:"case_status,omitempty"`
	Court               *Court           `gorm:"foreignKey:CourtID;references:CourtID" json:"court,omitempty"`
	Complainants        []ComplainantDetails `gorm:"foreignKey:CaseMasterID" json:"complainants,omitempty"`
	Victims             []Victim         `gorm:"foreignKey:CaseMasterID" json:"victims,omitempty"`
	AccusedList         []Accused        `gorm:"foreignKey:CaseMasterID" json:"accused_list,omitempty"`
	Arrests             []ArrestSurrender `gorm:"foreignKey:CaseMasterID" json:"arrests,omitempty"`
	ActsAssociated      []ActSectionAssociation `gorm:"foreignKey:CaseMasterID" json:"acts_associated,omitempty"`
	Chargesheet         *ChargesheetDetails `gorm:"foreignKey:CaseMasterID" json:"chargesheet,omitempty"`
	OccuranceTime       *Inv_OccuranceTime `gorm:"foreignKey:CaseMasterID" json:"occurance_time,omitempty"`
}

func (CaseMaster) TableName() string { return "CaseMaster" }

type ComplainantDetails struct {
	ComplainantID   int              `gorm:"primaryKey;autoIncrement;column:ComplainantID" json:"complainant_id"`
	CaseMasterID    int              `gorm:"column:CaseMasterID" json:"case_master_id"`
	ComplainantName string           `gorm:"column:ComplainantName;type:varchar(150)" json:"complainant_name"`
	AgeYear         int              `gorm:"column:AgeYear" json:"age_year"`
	OccupationID    int              `gorm:"column:OccupationID" json:"occupation_id"`
	ReligionID      int              `gorm:"column:ReligionID" json:"religion_id"`
	CasteID         int              `gorm:"column:CasteID" json:"caste_id"`
	GenderID        int              `gorm:"column:GenderID" json:"gender_id"`
	Occupation      *OccupationMaster `gorm:"foreignKey:OccupationID;references:OccupationID" json:"occupation,omitempty"`
	Religion        *ReligionMaster  `gorm:"foreignKey:ReligionID;references:ReligionID" json:"religion,omitempty"`
	Caste           *CasteMaster     `gorm:"foreignKey:CasteID;references:CasteMasterID" json:"caste,omitempty"`
}

func (ComplainantDetails) TableName() string { return "ComplainantDetails" }

type ActSectionAssociation struct {
	CaseMasterID   int    `gorm:"primaryKey;column:CaseMasterID" json:"case_master_id"`
	ActID          string `gorm:"primaryKey;column:ActID;type:varchar(50)" json:"act_id"`
	SectionID      string `gorm:"primaryKey;column:SectionID;type:varchar(50)" json:"section_code"`
	ActOrderID     int    `gorm:"column:ActOrderID" json:"act_order_id"`
	SectionOrderID int    `gorm:"column:SectionOrderID" json:"section_order_id"`
	Act            Act    `gorm:"foreignKey:ActID;references:ActCode" json:"act,omitempty"`
}

func (ActSectionAssociation) TableName() string { return "ActSectionAssociation" }

type Victim struct {
	VictimMasterID int    `gorm:"primaryKey;autoIncrement;column:VictimMasterID" json:"victim_master_id"`
	CaseMasterID   int    `gorm:"column:CaseMasterID" json:"case_master_id"`
	VictimName     string `gorm:"column:VictimName;type:varchar(150)" json:"victim_name"`
	AgeYear        int    `gorm:"column:AgeYear" json:"age_year"`
	GenderID       int    `gorm:"column:GenderID" json:"gender_id"`
	VictimPolice   string `gorm:"column:VictimPolice;type:varchar(10)" json:"victim_police"` // "1" = yes, "0" = no
}

func (Victim) TableName() string { return "Victim" }

type Accused struct {
	AccusedMasterID int    `gorm:"primaryKey;autoIncrement;column:AccusedMasterID" json:"accused_master_id"`
	CaseMasterID    int    `gorm:"column:CaseMasterID" json:"case_master_id"`
	AccusedName     string `gorm:"column:AccusedName;type:varchar(150)" json:"accused_name"`
	AgeYear         int    `gorm:"column:AgeYear" json:"age_year"`
	GenderID        int    `gorm:"column:GenderID" json:"gender_id"`
	PersonID        string `gorm:"column:PersonID;type:varchar(10)" json:"person_code"` // A1, A2, A3...
}

func (Accused) TableName() string { return "Accused" }

type ArrestSurrender struct {
	ArrestSurrenderID         int             `gorm:"primaryKey;autoIncrement;column:ArrestSurrenderID" json:"arrest_surrender_id"`
	CaseMasterID              int             `gorm:"column:CaseMasterID" json:"case_master_id"`
	ArrestSurrenderTypeID     int             `gorm:"column:ArrestSurrenderTypeID" json:"arrest_surrender_type_id"`
	ArrestSurrenderDate       time.Time       `gorm:"column:ArrestSurrenderDate;type:date" json:"arrest_surrender_date"`
	ArrestSurrenderStateId    int             `gorm:"column:ArrestSurrenderStateId" json:"arrest_surrender_state_id"`
	ArrestSurrenderDistrictId int             `gorm:"column:ArrestSurrenderDistrictId" json:"arrest_surrender_district_id"`
	PoliceStationID           int             `gorm:"column:PoliceStationID" json:"police_station_id"`
	IOID                      int             `gorm:"column:IOID" json:"io_id"`
	CourtID                   int             `gorm:"column:CourtID" json:"court_id"`
	AccusedMasterID           int             `gorm:"column:AccusedMasterID" json:"accused_master_id"`
	IsAccused                 bool            `gorm:"column:IsAccused;type:bit" json:"is_accused"`
	IsComplainantAccused      bool            `gorm:"column:IsComplainantAccused;type:bit" json:"is_complainant_accused"`
	State                     *State          `gorm:"foreignKey:ArrestSurrenderStateId;references:StateID" json:"state,omitempty"`
	District                  *District       `gorm:"foreignKey:ArrestSurrenderDistrictId;references:DistrictID" json:"district,omitempty"`
	PoliceStation             *Unit           `gorm:"foreignKey:PoliceStationID;references:UnitID" json:"police_station,omitempty"`
	InvestigatingOfficer      *Employee       `gorm:"foreignKey:IOID;references:EmployeeID" json:"investigating_officer,omitempty"`
	Court                     *Court          `gorm:"foreignKey:CourtID;references:CourtID" json:"court,omitempty"`
	Accused                   *Accused        `gorm:"foreignKey:AccusedMasterID;references:AccusedMasterID" json:"accused,omitempty"`
	AccusedLinks              []Accused       `gorm:"many2many:inv_arrestsurrenderaccused;foreignKey:ArrestSurrenderID;joinForeignKey:ArrestSurrenderID;References:AccusedMasterID;joinReferences:AccusedMasterID" json:"accused_links,omitempty"`
}

func (ArrestSurrender) TableName() string { return "ArrestSurrender" }

type InvArrestSurrenderAccused struct {
	ArrestSurrenderID int  `gorm:"primaryKey;column:ArrestSurrenderID" json:"arrest_surrender_id"`
	AccusedMasterID   int  `gorm:"primaryKey;column:AccusedMasterID" json:"accused_master_id"`
	IsPrimary         bool `gorm:"column:is_primary;type:bit;default:false" json:"is_primary"`
}

func (InvArrestSurrenderAccused) TableName() string { return "inv_arrestsurrenderaccused" }

type ChargesheetDetails struct {
	CSID           int       `gorm:"primaryKey;autoIncrement;column:CSID" json:"cs_id"`
	CaseMasterID   int       `gorm:"column:CaseMasterID;uniqueIndex" json:"case_master_id"`
	CsDate         time.Time `gorm:"column:csdate;type:datetime" json:"cs_date"`
	CsType         string    `gorm:"column:cstype;type:char(1)" json:"cs_type"` // A -> Chargesheet, B -> False Case, C -> Undetected
	PolicePersonID int       `gorm:"column:PolicePersonID" json:"police_person_id"`
	PolicePerson   *Employee `gorm:"foreignKey:PolicePersonID;references:EmployeeID" json:"police_person,omitempty"`
}

func (ChargesheetDetails) TableName() string { return "ChargesheetDetails" }

// --- SPATIO-TEMPORAL EXTENSION TABLE ---

type Inv_OccuranceTime struct {
	OccurrenceID       int       `gorm:"primaryKey;autoIncrement;column:OccurrenceID" json:"occurrence_id"`
	CaseMasterID       int       `gorm:"column:CaseMasterID;uniqueIndex" json:"case_master_id"`
	IncidentFromTs     time.Time `gorm:"column:IncidentFromTs;type:datetime" json:"incident_from_ts"`
	IncidentToTs       time.Time `gorm:"column:IncidentToTs;type:datetime" json:"incident_to_ts"`
	InfoReceivedPSTs   time.Time `gorm:"column:InfoReceivedPSTs;type:datetime" json:"info_received_ps_ts"`
	Latitude           float64   `gorm:"column:latitude;type:decimal(9,6)" json:"latitude"`
	Longitude          float64   `gorm:"column:longitude;type:decimal(9,6)" json:"longitude"`
	H3Cell             string    `gorm:"column:h3_cell;type:varchar(32)" json:"h3_cell"`
	AddressText        string    `gorm:"column:address_text;type:text" json:"address_text"`
	CreatedAt          time.Time `gorm:"column:created_at;type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (Inv_OccuranceTime) TableName() string { return "Inv_OccuranceTime" }

// --- AUTHENTICATION HELPERS ---

type UserCredentials struct {
	EmployeeID   int       `gorm:"primaryKey;column:EmployeeID" json:"employee_id"`
	PasswordHash string    `gorm:"column:PasswordHash;type:varchar(255)" json:"-"`
	Employee     Employee  `gorm:"foreignKey:EmployeeID;references:EmployeeID" json:"employee,omitempty"`
}

func (UserCredentials) TableName() string { return "UserCredentials" }

// --- AUXILIARY CHAT & AUDIT ENTITIES ---

type CaseDocument struct {
	DocumentID   int       `gorm:"primaryKey;autoIncrement;column:DocumentID" json:"document_id"`
	CaseMasterID int       `gorm:"column:CaseMasterID" json:"case_master_id"`
	DocumentType string    `gorm:"column:document_type;type:varchar(50)" json:"document_type"`
	StorageURI   string    `gorm:"column:storage_uri;type:text" json:"storage_uri"`
	SHA256       string    `gorm:"column:sha256;type:char(64)" json:"sha256"`
	LanguageCode string    `gorm:"column:language_code;type:varchar(16)" json:"language_code"`
	PiiLevel     string    `gorm:"column:pii_level;type:varchar(20)" json:"pii_level"`
	CreatedBy    int       `gorm:"column:created_by" json:"created_by"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
	Creator      *Employee `gorm:"foreignKey:CreatedBy;references:EmployeeID" json:"creator,omitempty"`
}

func (CaseDocument) TableName() string { return "CaseDocument" }

type ConversationSession struct {
	SessionID    string             `gorm:"primaryKey;column:SessionID;type:varchar(64)" json:"session_id"`
	UserID       int                `gorm:"column:UserID;index" json:"user_id"`
	CaseMasterID *int               `gorm:"column:CaseMasterID" json:"case_master_id,omitempty"`
	ContextJSON  string             `gorm:"column:ContextJSON;type:text" json:"context_json"` // JSON string
	CreatedAt    time.Time          `gorm:"column:created_at;type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
	User         *Employee          `gorm:"foreignKey:UserID;references:EmployeeID" json:"user,omitempty"`
	Case         *CaseMaster        `gorm:"foreignKey:CaseMasterID;references:CaseMasterID" json:"case,omitempty"`
	Turns        []ConversationTurn `gorm:"foreignKey:SessionID" json:"turns,omitempty"`
}

func (ConversationSession) TableName() string { return "ConversationSession" }

type ConversationTurn struct {
	TurnID      string    `gorm:"primaryKey;column:TurnID;type:varchar(64)" json:"turn_id"`
	SessionID   string    `gorm:"column:SessionID;type:varchar(64)" json:"session_id"`
	Speaker     string    `gorm:"column:speaker;type:varchar(20)" json:"speaker"` // user, bot
	Content     string    `gorm:"column:content;type:text" json:"content"`
	CitationJSON string   `gorm:"column:citation_json;type:text" json:"citation_json"` // JSON string
	CreatedAt   time.Time `gorm:"column:created_at;type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (ConversationTurn) TableName() string { return "ConversationTurn" }

type AuditEvent struct {
	AuditID    string    `gorm:"primaryKey;column:AuditID;type:varchar(64)" json:"audit_id"`
	Actor      string    `gorm:"column:actor;type:varchar(100)" json:"actor"` // KGID or username
	Action     string    `gorm:"column:action;type:varchar(100)" json:"action"`
	Resource   string    `gorm:"column:resource;type:varchar(255)" json:"resource"`
	BeforeHash string    `gorm:"column:before_hash;type:varchar(64)" json:"before_hash"`
	AfterHash  string    `gorm:"column:after_hash;type:varchar(64)" json:"after_hash"`
	RequestID  string    `gorm:"column:request_id;type:varchar(64)" json:"request_id"`
	TraceID    string    `gorm:"column:trace_id;type:varchar(64)" json:"trace_id"`
	CreatedAt  time.Time `gorm:"column:created_at;type:datetime;default:CURRENT_TIMESTAMP;index" json:"created_at"`
}

func (AuditEvent) TableName() string { return "AuditEvent" }

type RefreshToken struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	EmployeeID int       `gorm:"column:EmployeeID;index" json:"employee_id"`
	Token      string    `gorm:"type:varchar(255);uniqueIndex" json:"token"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

func (RefreshToken) TableName() string { return "RefreshToken" }
