package publish

import (
	"errors"
	"strings"
	"testing"

	"github.com/hishamkaram/gismanager/internal/errs"
)

// fullConfig returns a Manager with every required field
// populated to a sane non-zero value. Tests start from this and
// blank out one field at a time to exercise Validate's error paths.
func fullConfig() *Manager {
	return &Manager{
		Geoserver: GeoserverConfig{
			ServerURL:     "http://localhost:8080/geoserver",
			Username:      "admin",
			Password:      "geoserver",
			WorkspaceName: "golang",
		},
		Datastore: DatastoreConfig{
			Host:   "postgis",
			Port:   5432,
			DBName: "gis",
			DBUser: "golang",
			DBPass: "golang",
			Name:   "gismanager_data",
		},
		Source: SourceConfig{Path: "../testdata"},
	}
}

func TestManagerConfig_Validate_HappyPath(t *testing.T) {
	if err := fullConfig().Validate(); err != nil {
		t.Fatalf("Validate() on full config: want nil, got %v", err)
	}
}

func TestManagerConfig_Validate_RejectsEachMissingField(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*Manager)
		wantInMsg string
	}{
		{"missing geoserver.url", func(c *Manager) { c.Geoserver.ServerURL = "" }, "geoserver.url is required"},
		{"missing geoserver.workspace", func(c *Manager) { c.Geoserver.WorkspaceName = "" }, "geoserver.workspace is required"},
		{"missing geoserver.username", func(c *Manager) { c.Geoserver.Username = "" }, "geoserver.username is required"},
		{"missing geoserver.password", func(c *Manager) { c.Geoserver.Password = "" }, "geoserver.password is required"},
		{"missing datastore.host", func(c *Manager) { c.Datastore.Host = "" }, "datastore.host is required"},
		{"zero datastore.port", func(c *Manager) { c.Datastore.Port = 0 }, "datastore.port is required"},
		{"missing datastore.database", func(c *Manager) { c.Datastore.DBName = "" }, "datastore.database is required"},
		{"missing datastore.username", func(c *Manager) { c.Datastore.DBUser = "" }, "datastore.username is required"},
		{"missing datastore.password", func(c *Manager) { c.Datastore.DBPass = "" }, "datastore.password is required"},
		{"missing datastore.name", func(c *Manager) { c.Datastore.Name = "" }, "datastore.name is required"},
		{"missing source.path", func(c *Manager) { c.Source.Path = "" }, "source.path is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := fullConfig()
			tc.mutate(c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("Validate(): want error, got nil")
			}
			if !errors.Is(err, errs.ErrConfigInvalid) {
				t.Errorf("errors.Is(err, errs.ErrConfigInvalid) = false; want true. err=%v", err)
			}
			if !strings.Contains(err.Error(), tc.wantInMsg) {
				t.Errorf("error message missing %q\n got: %v", tc.wantInMsg, err)
			}
			var gerr *errs.GISError
			if !errors.As(err, &gerr) {
				t.Fatalf("errors.As to *errs.GISError: no match")
			}
			if gerr.Op != "Validate" {
				t.Errorf("Op = %q; want Validate", gerr.Op)
			}
		})
	}
}

// TestManagerConfig_Validate_AggregatesAllProblems confirms multiple
// missing fields surface in a single error envelope rather than the
// caller having to re-run Validate to find each one.
func TestManagerConfig_Validate_AggregatesAllProblems(t *testing.T) {
	c := fullConfig()
	c.Geoserver.ServerURL = ""
	c.Datastore.Host = ""
	c.Source.Path = ""
	err := c.Validate()
	if err == nil {
		t.Fatal("want aggregated error, got nil")
	}
	for _, want := range []string{
		"geoserver.url is required",
		"datastore.host is required",
		"source.path is required",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("aggregated error missing %q; got: %v", want, err)
		}
	}
}

func TestManagerConfig_ExpandEnv(t *testing.T) {
	t.Setenv("GISMGR_TEST_GS_URL", "http://demo:8080/geoserver")
	t.Setenv("GISMGR_TEST_GS_USER", "demo-user")
	t.Setenv("GISMGR_TEST_GS_PASS", "demo-pass")
	t.Setenv("GISMGR_TEST_PG_HOST", "demo-pg")
	t.Setenv("GISMGR_TEST_PG_USER", "demo-pg-user")
	t.Setenv("GISMGR_TEST_PG_PASS", "demo-pg-pass")
	t.Setenv("GISMGR_TEST_SRC_PATH", "/srv/data")

	c := &Manager{
		Geoserver: GeoserverConfig{
			ServerURL: "${GISMGR_TEST_GS_URL}",
			Username:  "${GISMGR_TEST_GS_USER}",
			Password:  "${GISMGR_TEST_GS_PASS}",
		},
		Datastore: DatastoreConfig{
			Host:   "${GISMGR_TEST_PG_HOST}",
			DBUser: "${GISMGR_TEST_PG_USER}",
			DBPass: "${GISMGR_TEST_PG_PASS}",
		},
		Source: SourceConfig{Path: "${GISMGR_TEST_SRC_PATH}"},
	}
	c.expandEnv()

	if c.Geoserver.ServerURL != "http://demo:8080/geoserver" {
		t.Errorf("Geoserver.ServerURL = %q", c.Geoserver.ServerURL)
	}
	if c.Geoserver.Username != "demo-user" {
		t.Errorf("Geoserver.Username = %q", c.Geoserver.Username)
	}
	if c.Datastore.DBPass != "demo-pg-pass" {
		t.Errorf("Datastore.DBPass = %q", c.Datastore.DBPass)
	}
	if c.Source.Path != "/srv/data" {
		t.Errorf("Source.Path = %q", c.Source.Path)
	}
}

// TestManagerConfig_ExpandEnv_UnsetVarBecomesEmpty locks in the
// os.ExpandEnv contract: unset variables resolve to empty strings,
// which Validate then catches with a useful field-name error.
func TestManagerConfig_ExpandEnv_UnsetVarBecomesEmpty(t *testing.T) {
	t.Setenv("GISMGR_TEST_REAL", "real-value")

	c := &Manager{
		Geoserver: GeoserverConfig{
			ServerURL: "${GISMGR_TEST_REAL}",
			Password:  "${GISMGR_TEST_DEFINITELY_UNSET_xyzzy}",
		},
	}
	c.expandEnv()

	if c.Geoserver.ServerURL != "real-value" {
		t.Errorf("set var: ServerURL = %q; want real-value", c.Geoserver.ServerURL)
	}
	if c.Geoserver.Password != "" {
		t.Errorf("unset var: Password = %q; want empty (ExpandEnv default)", c.Geoserver.Password)
	}
}

func TestFromConfig_RejectsMissingFields(t *testing.T) {
	mgr, err := FromConfig("../testdata/test_config_missing.yml")
	if mgr != nil {
		t.Errorf("manager = %v; want nil on incomplete config", mgr)
	}
	if err == nil {
		t.Fatal("err = nil; want validation failure")
	}
	if !errors.Is(err, errs.ErrConfigInvalid) {
		t.Errorf("errors.Is(err, errs.ErrConfigInvalid) = false; want true. err=%v", err)
	}
	// The aggregated error should call out every missing field.
	for _, want := range []string{
		"geoserver.password is required",
		"datastore.host is required",
		"datastore.username is required",
		"source.path is required",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error message missing %q; got: %v", want, err)
		}
	}
}

func TestFromConfig_ExpandsEnvVars(t *testing.T) {
	t.Setenv("GISMGR_TEST_GS_URL", "http://gs.demo:8080/geoserver")
	t.Setenv("GISMGR_TEST_GS_USER", "demo-admin")
	t.Setenv("GISMGR_TEST_GS_PASS", "demo-secret")
	t.Setenv("GISMGR_TEST_PG_HOST", "demo-postgis")
	t.Setenv("GISMGR_TEST_PG_USER", "demo-pg-user")
	t.Setenv("GISMGR_TEST_PG_PASS", "demo-pg-secret")
	t.Setenv("GISMGR_TEST_SRC_PATH", "../testdata")

	mgr, err := FromConfig("../testdata/test_config_envvars.yml")
	if err != nil {
		t.Fatalf("FromConfig: %v", err)
	}
	if got := mgr.Geoserver.ServerURL; got != "http://gs.demo:8080/geoserver" {
		t.Errorf("Geoserver.ServerURL = %q", got)
	}
	if got := mgr.Geoserver.Password; got != "demo-secret" {
		t.Errorf("Geoserver.Password = %q", got)
	}
	if got := mgr.Datastore.Host; got != "demo-postgis" {
		t.Errorf("Datastore.Host = %q", got)
	}
	if got := mgr.Datastore.DBPass; got != "demo-pg-secret" {
		t.Errorf("Datastore.DBPass = %q", got)
	}
	if got := mgr.Source.Path; got != "../testdata" {
		t.Errorf("Source.Path = %q", got)
	}
}
