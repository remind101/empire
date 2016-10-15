package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/remind101/empire/pkg/heroku"
)

var cmdApps = &Command{
	Run:      runApps,
	Usage:    "apps [<name>...]",
	Category: "app",
	Short:    "list apps",
	Long: `
Lists apps. Shows the app name, owner, and last release time (or
time the app was created, if it's never been released).

Examples:

    $ emp apps
    myapp     user@test.com         us  Jan 2 12:34
    myapp-eu  user@longdomainname…  eu  Jan 2 12:35

    $ emp apps myapp
    myapp  user@test.com  us  Jan 2 12:34
`,
}

func init() {
	cmdApps.Flag.StringVarP(&flagOrgName, "org", "o", "", "organization name")
}

func runApps(cmd *Command, names []string) {
	w := tabwriter.NewWriter(os.Stdout, 1, 2, 2, ' ', 0)
	defer w.Flush()
	var apps []hkapp
	if len(names) == 0 {
		var err error
		apps, err = getAppList(flagOrgName)
		must(err)
	} else {
		appch := make(chan *heroku.App, len(names))
		errch := make(chan error, len(names))
		for _, name := range names {
			if name == "" {
				appch <- nil
			} else {
				go func(appname string) {
					if app, err := client.AppInfo(appname); err != nil {
						errch <- err
					} else {
						appch <- app
					}
				}(name)
			}
		}
		for _ = range names {
			select {
			case err := <-errch:
				printFatal(err.Error())
			case app := <-appch:
				if app != nil {
					apps = append(apps, fromApp(*app))
				}
			}
		}
	}
	printAppList(w, apps)
}

func getAppList(orgName string) ([]hkapp, error) {
	if orgName != "" {
		apps, err := client.OrganizationAppListForOrganization(orgName, &heroku.ListRange{Field: "name", Max: 1000})
		if err != nil {
			return nil, err
		}
		return fromOrgApps(apps), nil
	}

	apps, err := client.AppList(&heroku.ListRange{Field: "name", Max: 1000})
	if err != nil {
		return nil, err
	}
	return fromApps(apps), nil
}

func printAppList(w io.Writer, apps []hkapp) {
	sort.Sort(appsByName(apps))
	abbrevEmailApps(apps)
	for _, a := range apps {
		if a.Name != "" {
			listApp(w, a)
		}
	}
}

func abbrevEmailApps(apps []hkapp) {
	domains := make(map[string]int)
	for _, a := range apps {
		if a.Organization != "" {
			parts := strings.SplitN(a.OwnerEmail, "@", 2)
			if len(parts) == 2 {
				domains["@"+parts[1]]++
			}
		}
	}
	smax, nmax := "", 0
	for s, n := range domains {
		if n > nmax {
			smax = s
			nmax = n
		}
	}
	for i := range apps {
		if apps[i].Organization != "" {
			// reference the app directly in the slice so we're not modifying a copy
			if strings.HasSuffix(apps[i].OwnerEmail, smax) {
				apps[i].OwnerEmail = apps[i].OwnerEmail[:len(apps[i].OwnerEmail)-len(smax)]
			}
		}
	}
}

func listApp(w io.Writer, a hkapp) {
	t := a.CreatedAt
	if a.ReleasedAt != nil {
		t = *a.ReleasedAt
	}
	orgOrEmail := a.Organization
	if orgOrEmail == "" {
		orgOrEmail = a.OwnerEmail
	}
	listRec(w,
		a.Name,
		fmtCerts(a.Certs),
		prettyTime{t},
	)
}

type fmtCerts map[string]string

func (certs fmtCerts) String() string {
	var p []string
	for process, cert := range certs {
		p = append(p, fmt.Sprintf("%s=%s", process, cert))
	}
	return strings.Join(p, ",")
}

type appsByName []hkapp

func (a appsByName) Len() int           { return len(a) }
func (a appsByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a appsByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

type hkapp struct {
	ArchivedAt                   *time.Time
	BuildpackProvidedDescription *string
	CreatedAt                    time.Time
	GitURL                       string
	Id                           string
	Maintenance                  bool
	Name                         string
	Organization                 string
	OwnerEmail                   string
	Region                       string
	ReleasedAt                   *time.Time
	RepoSize                     *int
	SlugSize                     *int
	Stack                        string
	UpdatedAt                    time.Time
	WebURL                       string
	Certs                        map[string]string
}

func fromApp(app heroku.App) (happ hkapp) {
	orgName := ""
	if strings.HasSuffix(app.Owner.Email, "@herokumanager.com") {
		orgName = strings.TrimSuffix(app.Owner.Email, "@herokumanager.com")
	}
	return hkapp{
		ArchivedAt:                   app.ArchivedAt,
		BuildpackProvidedDescription: app.BuildpackProvidedDescription,
		CreatedAt:                    app.CreatedAt,
		GitURL:                       app.GitURL,
		Id:                           app.Id,
		Maintenance:                  app.Maintenance,
		Name:                         app.Name,
		Organization:                 orgName,
		OwnerEmail:                   app.Owner.Email,
		Region:                       app.Region.Name,
		ReleasedAt:                   app.ReleasedAt,
		RepoSize:                     app.RepoSize,
		SlugSize:                     app.SlugSize,
		Stack:                        app.Stack.Name,
		UpdatedAt:                    app.UpdatedAt,
		WebURL:                       app.WebURL,
		Certs:                        app.Certs,
	}
}

func fromApps(apps []heroku.App) (happs []hkapp) {
	happs = make([]hkapp, len(apps))
	for i := range apps {
		happs[i] = fromApp(apps[i])
	}
	return
}

func fromOrgApp(oapp heroku.OrganizationApp) (happ hkapp) {
	orgName := ""
	if oapp.Organization != nil {
		orgName = oapp.Organization.Name
	}
	return hkapp{
		ArchivedAt:                   oapp.ArchivedAt,
		BuildpackProvidedDescription: oapp.BuildpackProvidedDescription,
		CreatedAt:                    oapp.CreatedAt,
		GitURL:                       oapp.GitURL,
		Id:                           oapp.Id,
		Maintenance:                  oapp.Maintenance,
		Name:                         oapp.Name,
		Organization:                 orgName,
		Region:                       oapp.Region.Name,
		ReleasedAt:                   oapp.ReleasedAt,
		RepoSize:                     oapp.RepoSize,
		SlugSize:                     oapp.SlugSize,
		Stack:                        oapp.Stack.Name,
		UpdatedAt:                    oapp.UpdatedAt,
		WebURL:                       oapp.WebURL,
	}
}

func fromOrgApps(oapps []heroku.OrganizationApp) (happs []hkapp) {
	happs = make([]hkapp, len(oapps))
	for i := range oapps {
		happs[i] = fromOrgApp(oapps[i])
	}
	return
}
