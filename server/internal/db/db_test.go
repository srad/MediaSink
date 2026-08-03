package db

import (
	"testing"

	"github.com/srad/mediasink/server/config"
)

// sqlDSN used to be spelled out twice inside Init, built from six separate
// os.Getenv calls. These cases pin the string both drivers receive.
func TestSQLDSN(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Cfg
		want string
	}{
		{
			name: "fully populated",
			cfg: config.Cfg{
				DBHost:     "db.internal",
				DBUser:     "mediasink",
				DBPassword: "hunter2",
				DBName:     "mediasink_prod",
				DBPort:     "5432",
			},
			want: "host=db.internal user=mediasink password=hunter2 dbname=mediasink_prod port=5432 sslmode=disable TimeZone=Europe/Berlin",
		},
		{
			// The old code produced exactly this for unset variables too; keeping the
			// behaviour means a misconfigured adapter still fails at connect time
			// with a readable DSN rather than somewhere else.
			name: "empty config",
			cfg:  config.Cfg{},
			want: "host= user= password= dbname= port= sslmode=disable TimeZone=Europe/Berlin",
		},
		{
			name: "password containing spaces is passed through verbatim",
			cfg: config.Cfg{
				DBHost:     "localhost",
				DBUser:     "u",
				DBPassword: "two words",
				DBName:     "n",
				DBPort:     "5432",
			},
			want: "host=localhost user=u password=two words dbname=n port=5432 sslmode=disable TimeZone=Europe/Berlin",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := sqlDSN(test.cfg); got != test.want {
				t.Errorf("sqlDSN() = %q, want %q", got, test.want)
			}
		})
	}
}

// The SQLite branch does not use sqlDSN; it appends pragmas to the file name. This
// pins that the two are not accidentally swapped.
func TestSQLDSNIgnoresTheSQLiteFileName(t *testing.T) {
	got := sqlDSN(config.Cfg{DbFileName: "/data/mediasink.db"})
	if want := "host= user= password= dbname= port= sslmode=disable TimeZone=Europe/Berlin"; got != want {
		t.Errorf("sqlDSN() = %q, want %q", got, want)
	}
}
