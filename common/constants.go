package common

import (
	"cloud.google.com/go/firestore"
	"time"
)

const EnvProjectID string = "PROJECT_ID"
const EnvOpencensusxProjectID string = "OPENCENSUSX_PROJECT_ID"
const EnvAuditLogTopic string = "AUDIT_LOG_TOPIC"
const EnvRetailerMessageTopic = "RETAILER_MESSAGE_TOPIC"
const EnvSiteMessageTopic = "SITE_MESSAGE_TOPIC"
const EnvSpokeMessageTopic = "SPOKE_MESSAGE_TOPIC"

const ServiceName string = "site-info-svc"
const RetailersCollection string = "site-info-retailers"
const SitesCollection string = "site-info-sites"
const RetailerAuditCollection string = "site-info-retailer-audit"
const SiteAuditCollection string = "site-info-site-audit"
const SpokesCollection = "site-info-spokes"
const SiteSpokeCollection = "site-info-site-spoke"

const RetailerIDPrefix string = "r"
const SiteIDPrefix string = "s"
const SpokeIDPrefix string = "p"
const RetailerPath string = "/retailers/"
const SitePath string = "/sites/"
const SpokePath string = "/spokes/"

const QueryParamDeactivated string = "deactivated"
const PathParamSiteID string = "site_id"
const PathParamRetailerID string = "retailer_id"
const PathParamSpokeID string = "spoke_id"
const PathParamDeactivate string = "deactivate"

const MaxRetryCount int = 3
const DefaultPageSize int = 25
const DataRetentionTime = time.Hour * 24 * 90 // 90 days
const CacheRetentionTime = time.Minute * 15   // 15 minutes
const ExpireTokenDuration float64 = 15        //in minutes
const APIVersionV1 string = "v1"

const HeaderAcceptVersion string = "Accept-Version"
const HeaderXCorrelationID string = "X-Correlation-ID"
const HeaderLastModified string = "Last-Modified"
const HeaderTimezone string = "timezone"
const HeaderLocation string = "Location"
const HeaderEtag string = "ETag"
const HeaderIfMatch string = "If-Match"
const HeaderPageToken string = "page_token"
const HeaderPageSize string = "page_size"
const HeaderRetailerID string = "retailer_id"
const HeaderNextPageToken string = "next_page_token"

const HeaderContentType string = "Content-Type"
const ContentTypeApplicationJSON string = "application/json"

const Name string = "name"
const ID string = "id"
const ETag string = "etag"
const RetailersSiteID string = "retailer_site_id"
const ChangedAt string = "changed_at"
const DeactivatedTime string = "deactivated_time"

const Status string = "status"
const SiteID string = "site_id"

const TimeParseFormat string = "2006-01-02 15:04:05 -0700 MST"

const Firestore string = "firestore"
const Disallowed string = "disallowed"
const Validate string = "validate"

const EntityRetailer string = "retailer"
const EntitySite string = "site"
const EntitySpoke string = "spoke"

const AuditTypeCreate string = "create"
const AuditTypeUpdate string = "update"
const AuditTypeDeactivate string = "deactivate"

const User string = "api@takeoff.com"

const RandomIDLength int = 5
const ColonSeparator string = "::::"
const Underscore = "_"
const True string = "true"
const RetailersEncryptionKey = "wC1wr8eci3fmWz" + EntityRetailer
const SitesEncryptionKey = "RCmJqUPtgW2NnjUq8m" + EntitySite
const SpokesEncryptionKey = "XgZXO5fFV2niFN2op" + EntitySpoke

const SortAscending = firestore.Asc
const SortDescending = firestore.Desc

const StatusDraft = "draft"
const StatusDeprecated = "deprecated"

const TimezoneAPIUrl = "https://maps.googleapis.com/maps/api/timezone/json"
const GoogleMapsAPIEnv = "GOOGLE_MAPS_API_KEY"
const LocationParam = "location"
const TimestampParam = "timestamp"
const APIKeyParam = "key"
const NameRegex = "^[a-zA-Z0-9]+(?:[. _-]*[a-zA-Z0-9]+)*$"
const MinNameLength = 5
const MaxNameLength = 128
const MinPageSize = 2
const MaxPageSize = 100
const ReturnError = -1
const OperatorEquals string = "=="
const OperatorIn string = "in"

const ChangeTypeCreate string = "create"
const ChangeTypeUpdate string = "update"
const ChangeTypeDelete string = "delete"

func GetMandatoryHeaders() []string {
	return []string{
		HeaderAcceptVersion, HeaderXCorrelationID,
	}
}

func GetSupportedVersions() []string {
	return []string{
		APIVersionV1,
	}
}

const StatusTransitionsCollection string = "site-info-status-transitions"
const SiteStatusTransitionsDocument string = "site-status-transitions"
