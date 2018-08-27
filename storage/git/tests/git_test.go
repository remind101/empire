package git_test

import (
	"context"
	"flag"
	"os"
	"testing"

	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"

	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/storage/git"
	"github.com/stretchr/testify/assert"
)

var (
	gitURL      = flag.String("test.git.url", "", "URL to clone/push")
	gitBasePath = flag.String("test.git.basepath", "apps/test", "Base path to commit to")
	gitRef      = flag.String("test.git.ref", "refs/heads/master", "Git ref to merge into")
)

// Does an complete functional test against a real GitHub repo.
func TestStorage(t *testing.T) {
	auth := newAuthMethod(t)
	s := git.NewStorage(auth)
	s.URL = *gitURL
	s.BasePath = *gitBasePath
	s.Ref = *gitRef
	s.Committer = &object.Signature{
		Name:  "Empire",
		Email: "empire@example.com",
	}

	app := &empire.App{
		Name:    "acme-inc",
		Version: 2,
		Environment: map[string]string{
			"FOO": "bar",
		},
		Image: &image.Image{Repository: "remind101/acme-inc"},
		Formation: empire.Formation{
			"web": {
				Command: empire.MustParseCommand("bash"),
			},
		},
	}

	user := &empire.User{
		Name: "ejholmes",
	}

	event := empire.DeployEvent{
		BaseEvent: empire.NewBaseEvent(user, "Some message included at deploy time"),
		App:       "acme-inc",
		Image:     "remind101/acme-inc:latest",
	}

	_, err := s.ReleasesCreate(context.Background(), os.Stdout, app, event)
	assert.NoError(t, err)

	//apps, err := s.Apps(empire.AppsQuery{})
	//assert.NoError(t, err)
	//assert.Equal(t, 1, len(apps))
	//assert.Equal(t, "acme-inc", apps[0].Name)

	//foundApp, err := s.AppsFind(empire.AppsQuery{Name: &app.Name})
	//assert.NoError(t, err)
	//assert.Equal(t, app.Image, foundApp.Image)
	//assert.Equal(t, app.Formation, foundApp.Formation)
	//assert.Equal(t, app.Environment, foundApp.Environment)
	//assert.Equal(t, app, foundApp)

	//releases, err := s.Releases(empire.ReleasesQuery{
	//App: &empire.App{
	//Name: "acme-inc",
	//},
	//})
	//assert.NoError(t, err)
	//assert.Equal(t, 1, len(releases))
	//assert.Equal(t, "Deployed remind101/acme-inc:latest to acme-inc", releases[0].Description)
}

func newAuthMethod(t testing.TB) transport.AuthMethod {
	auth, err := ssh.NewPublicKeys("git", []byte(privateKey), "")
	if err != nil {
		t.Fatal(err)
	}
	return auth
}

const publicKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC8b6alFsgPSnYW9SrL6ZCUk1BmZDVSK7Bp0f8Jk3+bGKOYwD6yaq3NRwNThFx1DRbplKSWKgp11YR2mKQsxYj+9pCBp0CPzeLEPm+NblE82vfdEE8PN6iWvq/U7bsREp/RwyndYB08IwHbLRySKkscDjsleRRdM1vYXR7+nnm4tnvkQkmXWV7pJNT6cAKYL6yU8s5sByAzHfUxpdxwNPcWgKApCsAkaWkey41DdyZPp1uqVoRUTRmKIFb2Sh5P6dDMCPuydUUTCFRkKeB/5OQRsDpZkvf2tHMZUnSR2g9XF00SIv8FRATI6YCH62CPs+xf++LcozoyTRSrks3NHgEx empire-git-test`

const privateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAvG+mpRbID0p2FvUqy+mQlJNQZmQ1UiuwadH/CZN/mxijmMA+
smqtzUcDU4RcdQ0W6ZSklioKddWEdpikLMWI/vaQgadAj83ixD5vjW5RPNr33RBP
Dzeolr6v1O27ERKf0cMp3WAdPCMB2y0ckipLHA47JXkUXTNb2F0e/p55uLZ75EJJ
l1le6STU+nACmC+slPLObAcgMx31MaXccDT3FoCgKQrAJGlpHsuNQ3cmT6dbqlaE
VE0ZiiBW9koeT+nQzAj7snVFEwhUZCngf+TkEbA6WZL39rRzGVJ0kdoPVxdNEiL/
BUQEyOmAh+tgj7PsX/vi3KM6Mk0Uq5LNzR4BMQIDAQABAoIBAC9bxD8jjZ5CMZGt
hlb/WYXtzDwfnUMMlebSj02B04VQWPVwA5Hlu86mtVBNKMuGQabs47yVzlE1G3iO
/lv+PWMO5tyfA1vg+8gkhPa/rg0kXz0g9y206FsSi7BxGx28V4xph2EK4e4WQkYq
fU7C8GOZtAXD+3k9Ci1DoiGOBLuGOv82h8AKM5JwMzdcrXfm+xBf/b8quRVuMUGj
fRwZsjmMml7X56ZKBtJIwCRywlYhQVkRUmSereWCUyxHzYPevMVyxUTekJS2eGCn
Cz+mtLwBR1shLhMU4hgAXtbsJbogFOKxuKR2KzAaKaanCnQyxCAuZ8846YD5RJzf
F9yCHAECgYEA42p6JBfeoU/nEOHoYkXYwZqD49EUP68lGWwQ3R9O2F1H18vCXYbi
doiAcbvmtwxfEFjjxdxEGNVhm8UznqUfFmYDxaushBpltXFEJSEbJoqXB9x+TQhL
VnWfgl6ufz6R7/5aNgXS0V4AzRrdnIY/ryIxKXt8+VfMCtgLab3ljcECgYEA1B7t
is96L+dMkVtBWImZ/+yJld2ChEDHGKt5gbFP1bd71YmXDbQtc81l6WLuoBVcWiQB
1hwhpSR1+1txBW5D/vwLGyTEqLx4ND9z+mZTrHttAxvasO026C93GDJYvBefWbVC
2bf7GV094KBZRPk5a+aTKPL5I0cqPXjD8s7EL3ECgYA5T6ggWHOeq4hL1OK/gWKy
x8HdM9+qfPUYhwdo8m7oT/m/BHajI08HGDdmcjgegVujvwMH5g6zQ3Mp0nTD0lEX
T/Y7Zpw/XnerVjQaN1GkUODm9kZOG30A+PuN21aWcqpMlQke/DC42rvQ9KhMvfhm
pnNFRP2lyN5/DRszNswXAQKBgGSf202bCLqKvb7CjGgktmI6YjYuo0h7tjbUuUo1
w8p9RQhSQc7lZx5lFHA3Vz2nhGzaYeig5bECb9lyUlYiLa1bASW6NnRZG6ilZS4m
tpC+6EKuRvrhPMe+JH/c+k0X46bZnsHLThmFKuJRDqfyljPCaJLnWBpNGxOYI1Qe
k+BxAoGAM0EOSEftGNliz1LtseTkybLIPqpQJQjhCkDXjhXibndeme9z1C1CXsVr
+ealRt9KCQbuS5LQRBjn9/2ubSCTEZpvIBzk0dvP/lmCZ0EOELBHBCBVhmaQIIOb
9OyVmPc+mPyw07LZrNPNuHdrZeSHcSKvv2Z89OnJobfDZKUYVOE=
-----END RSA PRIVATE KEY-----`
