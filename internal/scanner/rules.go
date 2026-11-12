package scanner

import "regexp"

type Rule struct {
	Pattern *regexp.Regexp
	Reason  string
	Risk    string
}

var DangerPatterns = []Rule{
	// Key-based patterns
	{regexp.MustCompile(`(?i)(aws_|gcp_|azure_)`), "Cloud provider credential", "CRITICAL"},
	{regexp.MustCompile(`(?i)(password|passwd|pwd)`), "Looks like a password field", "HIGH"},
	{regexp.MustCompile(`(?i)(secret|private_key|privatekey)`), "Looks like a secret/private key", "HIGH"},
	{regexp.MustCompile(`(?i)(api_key|apikey|access_key|stripe_|_key)`), "Looks like an API or service key", "HIGH"},
	{regexp.MustCompile(`(?i)(token|auth_token|jwt)`), "Looks like an auth token", "HIGH"},
	{regexp.MustCompile(`(?i)(database_url|db_url|mongo_uri)`), "Database connection string", "HIGH"},
}

var SecretValuePatterns = []Rule{
	// --- Cloud Providers ---
	{regexp.MustCompile(`AKIA[0-9A-Z]{16}`), "AWS Access Key ID", "CRITICAL"},
	{regexp.MustCompile(`(?i)aws_(s3|secret|access|key|token|id).*['\"][0-9a-zA-Z\/+]{40}['\"]`), "AWS Secret Access Key", "CRITICAL"},
	{regexp.MustCompile(`(?i)azure_.*['\"][0-9a-zA-Z]{32,128}['\"]`), "Azure Secret", "HIGH"},
	{regexp.MustCompile(`(?i)gcp_.*['\"][0-9a-zA-Z\-_]{32,128}['\"]`), "GCP Secret", "HIGH"},
	{regexp.MustCompile(`(?i)dop_v1_[a-z0-9]{64}`), "DigitalOcean Personal Access Token", "CRITICAL"},
	{regexp.MustCompile(`(?i)herokuan_.*['\"][0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}['\"]`), "Heroku API Key", "CRITICAL"},
	{regexp.MustCompile(`(?i)do_access_token.*['\"][0-9a-f]{64}['\"]`), "DigitalOcean Access Token", "CRITICAL"},

	// --- Payment Gateways ---
	{regexp.MustCompile(`sk_live_[0-9a-zA-Z]{24}`), "Stripe Live Secret Key", "CRITICAL"},
	{regexp.MustCompile(`rk_live_[0-9a-zA-Z]{24}`), "Stripe Live Restricted Key", "CRITICAL"},
	{regexp.MustCompile(`(?i)sq0idp-[0-9A-Za-z\\-_]{22}`), "Square Application ID", "HIGH"},
	{regexp.MustCompile(`(?i)sq0csp-[0-9A-Za-z\\-_]{43}`), "Square Access Token", "CRITICAL"},
	{regexp.MustCompile(`(?i)paypal_.*['\"][0-9a-zA-Z]{32,128}['\"]`), "PayPal Secret", "CRITICAL"},
	{regexp.MustCompile(`(?i)braintree_.*['\"][0-9a-z]{32}['\"]`), "Braintree Access Token", "CRITICAL"},

	// --- Developer Tools & VCS ---
	{regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`), "GitHub Personal Access Token", "CRITICAL"},
	{regexp.MustCompile(`github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59}`), "GitHub Fine-grained PAT", "CRITICAL"},
	{regexp.MustCompile(`gho_[a-zA-Z0-9]{36}`), "GitHub OAuth Access Token", "CRITICAL"},
	{regexp.MustCompile(`ghsr_[a-zA-Z0-9]{36}`), "GitHub Server-to-Server Token", "HIGH"},
	{regexp.MustCompile(`ghu_[a-zA-Z0-9]{36}`), "GitHub User-to-Server Token", "HIGH"},
	{regexp.MustCompile(`(?i)glpat-[0-9a-zA-Z\-]{20}`), "GitLab Personal Access Token", "CRITICAL"},
	{regexp.MustCompile(`(?i)bitbucket_.*['\"][0-9a-zA-Z]{32,128}['\"]`), "Bitbucket Secret", "CRITICAL"},
	{regexp.MustCompile(`(?i)travis_.*['\"][0-9a-zA-Z]{22}['\"]`), "Travis CI Access Token", "HIGH"},
	{regexp.MustCompile(`(?i)circleci_.*['\"][0-9a-f]{40}['\"]`), "CircleCI Personal Access Token", "CRITICAL"},

	// --- Messaging & Comms ---
	{regexp.MustCompile(`xox[baprs]-[0-9a-zA-Z]{10,48}`), "Slack Token", "CRITICAL"},
	{regexp.MustCompile(`https:\/\/hooks\.slack\.com\/services\/T[0-9a-zA-Z]{8}\/B[0-9a-zA-Z]{8}\/[0-9a-zA-Z]{24}`), "Slack Webhook URL", "HIGH"},
	{regexp.MustCompile(`(?i)discord_.*['\"][0-9a-zA-Z\._\-]{24,72}['\"]`), "Discord Token", "CRITICAL"},
	{regexp.MustCompile(`(?i)twilio_.*['\"][0-9a-f]{32}['\"]`), "Twilio API Key / Auth Token", "CRITICAL"},
	{regexp.MustCompile(`SG\.[0-9a-zA-Z\-_]{22}\.[0-9a-zA-Z\-_]{43}`), "SendGrid API Key", "CRITICAL"},
	{regexp.MustCompile(`key-[0-9a-zA-Z]{32}`), "Mailgun API Key", "CRITICAL"},
	{regexp.MustCompile(`(?i)messagebird_.*['\"][0-9a-zA-Z]{20,40}['\"]`), "MessageBird API Key", "HIGH"},

	// --- Databases & Infrastructure ---
	{regexp.MustCompile(`(?i)postgres(ql)?:\/\/.*:.*@.*:.*\/.*`), "PostgreSQL Connection String", "CRITICAL"},
	{regexp.MustCompile(`(?i)mongodb(\+srv)?:\/\/.*:.*@.*`), "MongoDB Connection String", "CRITICAL"},
	{regexp.MustCompile(`(?i)redis:\/\/.*:.*@.*`), "Redis Connection String", "CRITICAL"},
	{regexp.MustCompile(`(?i)mysql:\/\/.*:.*@.*`), "MySQL Connection String", "CRITICAL"},
	{regexp.MustCompile(`(?i)sqldb:\/\/.*:.*@.*`), "SQL Database Connection String", "CRITICAL"},
	{regexp.MustCompile(`(?i)elasticsearch_.*['\"][0-9a-zA-Z]{32,128}['\"]`), "Elasticsearch Secret", "HIGH"},

	// --- Social Media & Misc ---
	{regexp.MustCompile(`(?i)facebook_.*['\"][0-9a-f]{32}['\"]`), "Facebook App Secret", "CRITICAL"},
	{regexp.MustCompile(`(?i)twitter_.*['\"][0-9a-zA-Z]{35,44}['\"]`), "Twitter Secret", "CRITICAL"},
	{regexp.MustCompile(`(?i)instagram_.*['\"][0-9a-f]{32}['\"]`), "Instagram App Secret", "CRITICAL"},
	{regexp.MustCompile(`(?i)linkedin_.*['\"][0-9a-zA-Z]{16}['\"]`), "LinkedIn Client Secret", "CRITICAL"},
	{regexp.MustCompile(`(?i)AIza[0-9A-Za-z\\-_]{35}`), "Google API Key", "HIGH"},
	{regexp.MustCompile(`(?i)google_.*['\"][0-9a-zA-Z\-_]{24}['\"]`), "Google Secret", "HIGH"},

	// --- Security & Cryptography ---
	{regexp.MustCompile(`(?s)-----BEGIN (RSA|DSA|EC|OPENSSH|PRIVATE) KEY-----.*-----END [A-Z ]+ KEY-----`), "Private Key Block", "CRITICAL"},
	{regexp.MustCompile(`(?s)-----BEGIN PGP PRIVATE KEY BLOCK-----.*-----END PGP PRIVATE KEY BLOCK-----`), "PGP Private Key Block", "CRITICAL"},
	{regexp.MustCompile(`(?s)-----BEGIN CERTIFICATE-----.*-----END CERTIFICATE-----`), "Certificate Block", "HIGH"},
	{regexp.MustCompile(`ey[a-zA-Z0-9]{10,}\.ey[a-zA-Z0-9]{10,}\.[a-zA-Z0-9\-_]{10,}`), "JWT Token", "HIGH"},

	// --- Generic High Entropy Patterns (Fallback) ---
	{regexp.MustCompile(`[0-9a-f]{32,64}`), "Hexadecimal Hash / Key (Potential Secret)", "MEDIUM"},
	{regexp.MustCompile(`[0-9a-zA-Z\-_]{40,86}`), "Base64-like String (Potential Secret)", "MEDIUM"},

	// --- Additional SaaS & Services ---
	{regexp.MustCompile(`(?i)adobe_.*['\"][0-9a-f]{32}['\"]`), "Adobe Client Secret", "HIGH"},
	{regexp.MustCompile(`(?i)algolia_.*['\"][0-9a-f]{32}['\"]`), "Algolia API Key", "HIGH"},
	{regexp.MustCompile(`(?i)asana_.*['\"][0-9]\/[0-9]{16}:[a-zA-Z0-9]{32}['\"]`), "Asana Personal Access Token", "CRITICAL"},
	{regexp.MustCompile(`(?i)cloudflare_.*['\"][0-9a-f]{40}['\"]`), "Cloudflare API Key", "CRITICAL"},
	{regexp.MustCompile(`(?i)datadog_.*['\"][0-9a-f]{32}['\"]`), "Datadog API Key", "CRITICAL"},
	{regexp.MustCompile(`(?i)digitalocean_.*['\"][0-9a-f]{64}['\"]`), "DigitalOcean Token", "CRITICAL"},
	{regexp.MustCompile(`(?i)dropbox_.*['\"][a-z0-9]{15}['\"]`), "Dropbox Access Token", "CRITICAL"},
	{regexp.MustCompile(`(?i)firebase_.*['\"][0-9a-zA-Z\-_]{40}['\"]`), "Firebase Secret", "CRITICAL"},
	{regexp.MustCompile(`(?i)hubspot_.*['\"][0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}['\"]`), "HubSpot API Key", "CRITICAL"},
	{regexp.MustCompile(`(?i)mailchimp_.*['\"][0-9a-f]{32}-us[0-9]{1,2}['\"]`), "Mailchimp API Key", "CRITICAL"},
	{regexp.MustCompile(`(?i)newrelic_.*['\"][0-9a-f]{40}['\"]`), "New Relic API Key", "CRITICAL"},
	{regexp.MustCompile(`(?i)pagerduty_.*['\"][0-9a-zA-Z]{20}['\"]`), "PagerDuty API Key", "CRITICAL"},
	{regexp.MustCompile(`(?i)shopify_.*['\"][0-9a-f]{32}['\"]`), "Shopify Shared Secret", "CRITICAL"},
	{regexp.MustCompile(`(?i)slack_webhook_.*['\"]https:\/\/hooks\.slack\.com\/services\/[A-Z0-9\/]+['\"]`), "Slack Webhook URL", "HIGH"},
	{regexp.MustCompile(`(?i)splunk_.*['\"][0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}['\"]`), "Splunk Access Token", "HIGH"},
	{regexp.MustCompile(`(?i)zendesk_.*['\"][0-9a-zA-Z]{40}['\"]`), "Zendesk API Token", "CRITICAL"},

	// --- Specific Token Types ---
	{regexp.MustCompile(`(?i)aws_mws_.*['\"][0-9a-zA-Z]{32,128}['\"]`), "AWS MWS Auth Token", "CRITICAL"},
	{regexp.MustCompile(`(?i)aws_session_.*['\"][0-9a-zA-Z\/+]{128,512}['\"]`), "AWS Session Token", "HIGH"},
	{regexp.MustCompile(`(?i)google_oauth_.*['\"][0-9a-zA-Z\-_]{24}['\"]`), "Google OAuth Secret", "HIGH"},
	{regexp.MustCompile(`(?i)google_service_account_.*['\"][0-9a-zA-Z\-_]{32,128}['\"]`), "Google Service Account Key", "CRITICAL"},
	{regexp.MustCompile(`(?i)heroku_oauth_.*['\"][0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}['\"]`), "Heroku OAuth Token", "CRITICAL"},
	{regexp.MustCompile(`(?i)twilio_account_sid.*['\"]AC[0-9a-f]{32}['\"]`), "Twilio Account SID", "HIGH"},
	{regexp.MustCompile(`(?i)twilio_auth_token.*['\"][0-9a-f]{32}['\"]`), "Twilio Auth Token", "CRITICAL"},
	{regexp.MustCompile(`(?i)stripe_webhook_.*['\"][0-9a-zA-Z]{32,128}['\"]`), "Stripe Webhook Secret", "HIGH"},
	{regexp.MustCompile(`(?i)paypal_client_id.*['\"][0-9a-zA-Z]{32,128}['\"]`), "PayPal Client ID", "HIGH"},
	{regexp.MustCompile(`(?i)facebook_page_access_token.*['\"][0-9a-zA-Z]{128,512}['\"]`), "Facebook Page Access Token", "CRITICAL"},
	{regexp.MustCompile(`(?i)twitter_bearer_token.*['\"][0-9a-zA-Z%]{64,256}['\"]`), "Twitter Bearer Token", "HIGH"},
	{regexp.MustCompile(`(?i)github_app_client_id.*['\"]Iv1\.[0-9a-f]{16}['\"]`), "GitHub App Client ID", "HIGH"},
}

var WeakValues = []string{
	"password", "123456", "secret", "test", "admin", "changeme", "1234", "qwerty",
}

