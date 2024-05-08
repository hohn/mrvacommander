package types

type DownloadResponse struct {
	Repository           DownloadRepo `json:"repository"`
	AnalysisStatus       string       `json:"analysis_status"`
	ResultCount          int          `json:"result_count"`
	ArtifactSizeBytes    int          `json:"artifact_size_in_bytes"`
	DatabaseCommitSha    string       `json:"database_commit_sha"`
	SourceLocationPrefix string       `json:"source_location_prefix"`
	ArtifactURL          string       `json:"artifact_url"`
}

type DownloadRepo struct {
	ID       int    `json:"id"`
	NodeID   string `json:"node_id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
	Owner    Actor  `json:"owner"`

	HTMLURL          string `json:"html_url"`
	Description      string `json:"description"`
	Fork             bool   `json:"fork"`
	ForksURL         string `json:"forks_url"`
	KeysURL          string `json:"keys_url"`
	CollaboratorsURL string `json:"collaborators_url"`
	TeamsURL         string `json:"teams_url"`
	HooksURL         string `json:"hooks_url"`
	IssueEventsURL   string `json:"issue_events_url"`
	EventsURL        string `json:"events_url"`

	AssigneesURL string `json:"assignees_url"`
	BranchesURL  string `json:"branches_url"`
	TagsURL      string `json:"tags_url"`
	BlobsURL     string `json:"blobs_url"`
	GitTagsURL   string `json:"git_tags_url"`
	GitRefsURL   string `json:"git_refs_url"`
	TreesURL     string `json:"trees_url"`
	StatusesURL  string `json:"statuses_url"`
	LanguagesURL string `json:"languages_url"`

	StargazersURL   string `json:"stargazers_url"`
	ContributorsURL string `json:"contributors_url"`
	SubscribersURL  string `json:"subscribers_url"`
	SubscriptionURL string `json:"subscription_url"`

	CommitsURL       string `json:"commits_url"`
	GitCommitsURL    string `json:"git_commits_url"`
	CommentsURL      string `json:"comments_url"`
	IssueCommentURL  string `json:"issue_comment_url"`
	ContentsURL      string `json:"contents_url"`
	CompareURL       string `json:"compare_url"`
	MergesURL        string `json:"merges_url"`
	ArchiveURL       string `json:"archive_url"`
	DownloadsURL     string `json:"downloads_url"`
	IssuesURL        string `json:"issues_url"`
	PullsURL         string `json:"pulls_url"`
	MilestonesURL    string `json:"milestones_url"`
	NotificationsURL string `json:"notifications_url"`
	LabelsURL        string `json:"labels_url"`
	ReleasesURL      string `json:"releases_url"`
	DeploymentsURL   string `json:"deployments_url"`
}

type ControllerRepo struct {
	ID               int      `json:"id"`
	NodeID           string   `json:"node_id"`
	Name             string   `json:"name"`
	FullName         string   `json:"full_name"`
	Private          bool     `json:"private"`
	Owner            struct{} `json:"owner"`
	HTMLURL          string   `json:"html_url"`
	Description      string   `json:"description"`
	Fork             bool     `json:"fork"`
	ForksURL         string   `json:"forks_url"`
	KeysURL          string   `json:"keys_url"`
	CollaboratorsURL string   `json:"collaborators_url"`
	TeamsURL         string   `json:"teams_url"`
	HooksURL         string   `json:"hooks_url"`
	IssueEventsURL   string   `json:"issue_events_url"`
	EventsURL        string   `json:"events_url"`

	AssigneesURL string `json:"assignees_url"`
	BranchesURL  string `json:"branches_url"`
	TagsURL      string `json:"tags_url"`
	BlobsURL     string `json:"blobs_url"`
	GitTagsURL   string `json:"git_tags_url"`
	GitRefsURL   string `json:"git_refs_url"`
	TreesURL     string `json:"trees_url"`
	StatusesURL  string `json:"statuses_url"`
	LanguagesURL string `json:"languages_url"`

	StargazersURL   string `json:"stargazers_url"`
	ContributorsURL string `json:"contributors_url"`
	SubscribersURL  string `json:"subscribers_url"`
	SubscriptionURL string `json:"subscription_url"`

	CommitsURL       string `json:"commits_url"`
	GitCommitsURL    string `json:"git_commits_url"`
	CommentsURL      string `json:"comments_url"`
	IssueCommentURL  string `json:"issue_comment_url"`
	ContentsURL      string `json:"contents_url"`
	CompareURL       string `json:"compare_url"`
	MergesURL        string `json:"merges_url"`
	ArchiveURL       string `json:"archive_url"`
	DownloadsURL     string `json:"downloads_url"`
	IssuesURL        string `json:"issues_url"`
	PullsURL         string `json:"pulls_url"`
	MilestonesURL    string `json:"milestones_url"`
	NotificationsURL string `json:"notifications_url"`
	LabelsURL        string `json:"labels_url"`
	ReleasesURL      string `json:"releases_url"`
	DeploymentsURL   string `json:"deployments_url"`
}

type Actor struct {
	Login      string `json:"login"`
	ID         int    `json:"id"`
	NodeID     string `json:"node_id"`
	AvatarURL  string `json:"avatar_url"`
	GravatarID string `json:"gravatar_id"`

	URL          string `json:"url"`
	HTMLURL      string `json:"html_url"`
	FollowersURL string `json:"followers_url"`
	FollowingURL string `json:"following_url"`
	GistsURL     string `json:"gists_url"`

	StarredURL       string `json:"starred_url"`
	SubscriptionsURL string `json:"subscriptions_url"`
	OrganizationsURL string `json:"organizations_url"`
	ReposURL         string `json:"repos_url"`
	EventsURL        string `json:"events_url"`

	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
	SiteAdmin         bool   `json:"site_admin"`
}

type SkippedRepositories struct {
	AccessMismatchRepos AccessMismatchRepos `json:"access_mismatch_repos"`
	NotFoundRepos       NotFoundRepos       `json:"not_found_repos"`
	NoCodeqlDBRepos     NoCodeqlDBRepos     `json:"no_codeql_db_repos"`
	OverLimitRepos      OverLimitRepos      `json:"over_limit_repos"`
}

type ignored_repos struct {
	RepositoryCount int      `json:"repository_count"`
	Repositories    []string `json:"repositories"`
}

type AccessMismatchRepos struct {
	RepositoryCount int      `json:"repository_count"`
	Repositories    []string `json:"repositories"`
}

type NotFoundRepos struct {
	RepositoryCount     int      `json:"repository_count"`
	RepositoryFullNames []string `json:"repository_full_names"`
}

type NoCodeqlDBRepos struct {
	RepositoryCount int      `json:"repository_count"`
	Repositories    []string `json:"repositories"`
}

type OverLimitRepos struct {
	RepositoryCount int      `json:"repository_count"`
	Repositories    []string `json:"repositories"`
}

type SubmitResponse struct {
	ID                  int                 `json:"id"`
	ControllerRepo      ControllerRepo      `json:"controller_repo"`
	Actor               Actor               `json:"actor"`
	QueryLanguage       string              `json:"query_language"`
	QueryPackURL        string              `json:"query_pack_url"`
	CreatedAt           string              `json:"created_at"`
	UpdatedAt           string              `json:"updated_at"`
	Status              string              `json:"status"`
	SkippedRepositories SkippedRepositories `json:"skipped_repositories"`
}

type Repository struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	FullName        string `json:"full_name"`
	Private         bool   `json:"private"`
	StargazersCount int    `json:"stargazers_count"`
	UpdatedAt       string `json:"updated_at"`
}

type ScannedRepo struct {
	Repository        Repository `json:"repository"`
	AnalysisStatus    string     `json:"analysis_status"`
	ResultCount       int        `json:"result_count"`
	ArtifactSizeBytes int        `json:"artifact_size_in_bytes"`
}

type StatusResponse struct {
	SessionId            int                 `json:"id"`
	ControllerRepo       ControllerRepo      `json:"controller_repo"`
	Actor                Actor               `json:"actor"`
	QueryLanguage        string              `json:"query_language"`
	QueryPackURL         string              `json:"query_pack_url"`
	CreatedAt            string              `json:"created_at"`
	UpdatedAt            string              `json:"updated_at"`
	ActionsWorkflowRunID int                 `json:"actions_workflow_run_id"`
	Status               string              `json:"status"`
	ScannedRepositories  []ScannedRepo       `json:"scanned_repositories"`
	SkippedRepositories  SkippedRepositories `json:"skipped_repositories"`
}
