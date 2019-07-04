package kvstore

// TODO find a better way of documenting all DB keys in one place.
const (
	// Used by Jira Server to store OAuth1a temporary credentials.
	KeyPrefixTempOAuth1aCredentials = "oauth1_temp_cred_"

	// Used by Jira Cloud to store unconfirmed instances.
	KeyPrefixUnconfirmedUpstream = "unconfirmed_upstream_"

	KeyCurrentUpstream = "current_jira_instance"
	KeyKnownUpstreams  = "known_jira_instances"
	KeyPrefixUpstream  = "jira_instance_"
)
