package sourcegraph

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"

	"github.com/jingweno/ccat/Godeps/_workspace/src/sourcegraph.com/sourcegraph/go-sourcegraph/router"
)

// PeopleService communicates with the people-related endpoints in the
// Sourcegraph API.
type PeopleService interface {
	// Get gets a person. If an email is provided and it resolves to a
	// registered user, information about that user is
	// returned. Otherwise a transient person is created and returned.
	Get(person PersonSpec) (*Person, Response, error)
}

// peopleService implements PeopleService.
type peopleService struct {
	client *Client
}

var _ PeopleService = &peopleService{}

// A Person represents either a registered user or a committer to a
// repository (typically when their commit email can't be resolved to
// a user).
type Person struct {
	// PersonSpec is an identifier for the person. If the person was
	// resolved to a user, then both Login and UID are set. Otherwise
	// only Email is set, and it may be obfuscated (to protect
	// privacy).
	PersonSpec

	// FullName is the (possibly empty) full name of the person.
	FullName string

	// AvatarURL is the URL to the user's avatar image.
	AvatarURL string
}

// ShortName returns the person's Login if nonempty and otherwise
// returns the portion of Email before the '@'.
func (p *Person) ShortName() string {
	if p.Login != "" {
		return p.Login
	}
	at := strings.Index(p.Email, "@")
	if at == -1 {
		return "(anonymous)"
	}
	return p.Email[:at]
}

// Transient is true if this person was constructed on the fly and is
// not persisted or resolved to a Sourcegraph/GitHub/etc. user.
func (p *Person) Transient() bool { return p.UID == 0 }

// HasProfile is true if the person has a profile page on
// Sourcegraph. Transient users currently do not have profile pages.
func (p *Person) HasProfile() bool { return !p.Transient() }

// AvatarURLOfSize returns the URL to an avatar for the user with the
// given width (in pixels).
func (p *Person) AvatarURLOfSize(width int) string {
	return avatarURLOfSize(p.AvatarURL, width)
}

// PersonSpec specifies a person. At least one of Email, Login, and UID must be
// nonempty.
type PersonSpec struct {
	// Email is a person's email address. It may be obfuscated (to
	// protect privacy).
	Email string

	// Login is a user's login.
	Login string

	// UID is a user's UID.
	UID int
}

// PathComponent returns the URL path component that specifies the person.
func (s *PersonSpec) PathComponent() string {
	if s.Email != "" {
		return s.Email
	}
	if s.Login != "" {
		return s.Login
	}
	if s.UID > 0 {
		return "$" + strconv.Itoa(s.UID)
	}
	panic("empty PersonSpec")
}

func (s *PersonSpec) RouteVars() map[string]string {
	return map[string]string{"PersonSpec": s.PathComponent()}
}

// ParsePersonSpec parses a string generated by (*PersonSpec).String() and
// returns the equivalent PersonSpec struct.
func ParsePersonSpec(pathComponent string) (PersonSpec, error) {
	if strings.HasPrefix(pathComponent, "$") {
		uid, err := strconv.Atoi(pathComponent[1:])
		return PersonSpec{UID: uid}, err
	}
	if strings.Contains(pathComponent, "@") {
		return PersonSpec{Email: pathComponent}, nil
	}
	return PersonSpec{Login: pathComponent}, nil
}

func (s *peopleService) Get(spec PersonSpec) (*Person, Response, error) {
	url, err := s.client.URL(router.Person, spec.RouteVars(), nil)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	var person *Person
	resp, err := s.client.Do(req, &person)
	if err != nil {
		return nil, resp, err
	}

	return person, resp, nil
}

type PersonStatType string

type PersonStats map[PersonStatType]int

const (
	PersonStatAuthors            = "authors"
	PersonStatClients            = "clients"
	PersonStatOwnedRepos         = "owned-repos"
	PersonStatContributedToRepos = "contributed-to-repos"
	PersonStatDependencies       = "dependencies"
	PersonStatDependents         = "dependents"
	PersonStatDefs               = "defs"
	PersonStatExportedDefs       = "exported-defs"
)

func (x PersonStatType) Value() (driver.Value, error) {
	return string(x), nil
}

func (x *PersonStatType) Scan(v interface{}) error {
	if data, ok := v.([]byte); ok {
		*x = PersonStatType(data)
		return nil
	}
	return fmt.Errorf("%T.Scan failed: %v", x, v)
}

var _ PeopleService = &MockPeopleService{}
