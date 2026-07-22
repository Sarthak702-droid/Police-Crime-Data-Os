export interface ApiResponse<T> {
  success: boolean
  message: string
  data: T
  error?: string
}

export interface Officer {
  employee_id?: number
  EmployeeID?: number
  kgid?: string
  KGID?: string
  first_name?: string
  FirstName?: string
  rank?: string
  rank_hierarchy?: number
  designation?: string
  unit_id?: number
  UnitID?: number
  district_id?: number
  DistrictID?: number
}

export interface CaseRecord {
  CaseMasterID: number
  CrimeNo: string
  CrimeRegisteredDate: string
  BriefFacts: string
  CaseCategoryID: number
  GravityOffenceID: number
  CrimeMajorHeadID: number
  CrimeMinorHeadID: number
  PoliceStationID: number
  Latitude: number
  Longitude: number
  CurrentStatusID?: number
  IncidentFromDate?: string
  IncidentToDate?: string
  Complainants?: unknown[]
  Victims?: unknown[]
  AccusedList?: Array<Record<string, unknown>>
  ActsAssociated?: unknown[]
  Arrests?: unknown[]
  Documents?: unknown[]
}

export interface SearchResult {
  cases: CaseRecord[]
  total: number
  page: number
  limit: number
}

export interface PendingCase {
  case_master_id: number
  crime_no: string
  age_days: number
  priority_score: number
  missing_actions: string[]
}

export interface Readiness {
  case_master_id: number
  crime_no: string
  score: number
  band: string
  checks: Array<{ name: string; passed: boolean; weight: number; action?: string }>
  disclaimer: string
}

export interface SimilarCase {
  case_master_id: number
  crime_no: string
  registered_date: string
  score: number
  distance_km?: number
  reasons: string[]
}

export interface ChatResponse {
  answer?: string
  response?: string
  session_id?: string
  tool_used?: string
  confidence?: number
  evidence?: unknown
}

export interface ChatTurn {
  ChatTurnID?: number
  Speaker?: string
  Content?: string
  CreatedAt?: string
  speaker?: string
  content?: string
  created_at?: string
}
