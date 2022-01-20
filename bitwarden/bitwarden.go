package bitwarden

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/echocat/slf4g"
	"github.com/echocat/terraform-provider-bitwarden/utils"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	ErrNotLoggedIn   = errors.New("not logged in")
	ErrWrongSession  = errors.New("BW_SESSION either wrong or expired")
	ErrNoSuchItem    = errors.New("no such item")
	ErrItemNotUnique = errors.New("item not unique")

	DetailWrongSession = "BW_SESSION does contain a wrong or expired session token. Try either `bw unlock` (if already logged it) or `bw login` to acquire a new session token and set the content to BW_SESSION environment variable."
)

func NewBitwarden(session, executable string) (*Bitwarden, error) {
	if executable == "" {
		executable = os.Getenv("BW_CLI_EXECUTABLE")
	}
	if executable == "" {
		executable = "bw"
	}
	if session == "" {
		session = os.Getenv("BW_SESSION")
	}

	var resolvedExecutable []string
	if strings.ToLower(filepath.Ext(executable)) == ".js" {
		fi, err := os.Stat(executable)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve executable (%s): %w", executable, err)
		}
		if fi.IsDir() {
			return nil, fmt.Errorf("executable (%s) is not a file", executable)
		}
		node, err := lookupExecutable("node")
		if err != nil {
			return nil, fmt.Errorf("cannot find node executable (%s): %w", executable, err)
		}
		resolvedExecutable = []string{node, executable}
	} else {
		v, err := lookupExecutable(executable)
		if err != nil {
			return nil, fmt.Errorf("cannot find bitwarden executable (%s): %w", executable, err)
		}
		resolvedExecutable = []string{v}
	}

	return &Bitwarden{
		executable:       resolvedExecutable,
		session:          session,
		legacyAttachment: true,
	}, nil
}

type Bitwarden struct {
	executable       []string
	session          string
	legacyAttachment bool
}

func (this *Bitwarden) Session() string {
	return this.session
}

func (this *Bitwarden) Test() (bool, error) {
	status, _, err := this.Status()
	if err != nil {
		return false, err
	}
	return status.IsUsable(), nil
}

func (this *Bitwarden) Status() (status Status, user string, err error) {
	v := struct {
		User   string `json:"userEmail"`
		Status Status `json:"status"`
	}{}
	err = this.ExecuteAndUnmarshal(nil, &v, "status")
	if errors.Unwrap(err) == ErrWrongSession {
		return 0, "", nil
	}
	if err != nil {
		return 0, "", err
	}
	return v.Status, v.User, nil
}

func (this *Bitwarden) Unlock(onlyIfRequired bool) error {
	status, username, err := this.Status()
	if err != nil {
		return err
	}
	if status == StatusUnauthenticated || username == "" {
		return ErrNotLoggedIn
	}
	if !status.IsUsable() || !onlyIfRequired {
		if err := this.unlock(username); err != nil {
			return err
		}
	}
	return nil
}

func (this *Bitwarden) unlock(username string) error {
	mp, err := this.readMasterPassword(username)
	if err != nil {
		return err
	}
	session, stderr, err := this.ExecuteDirect(func(cmd *exec.Cmd) {
		cmd.Env = append(cmd.Env, "BW_MASTER_PASSWORD="+mp)
	}, "unlock", "--raw", "--passwordenv", "BW_MASTER_PASSWORD")
	if strings.Contains(stderr, "Invalid master password.") {
		return fmt.Errorf("illegal master password")
	}
	if err != nil {
		return err
	}
	this.session = string(session)
	return nil
}

func (this *Bitwarden) readMasterPassword(username string) (string, error) {
	prompt := func() {
		fmt.Printf("Bitwarden Master password (%s): ", username)
	}
	prompt()
	result, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err == nil {
		fmt.Println()
		return string(result), nil
	}
	if err != nil && err.Error() != "The handle is invalid." {
		return "", err
	}

	fmt.Print("\rWARNING! Password will be visible in the console. Hide your screen and close the console afterwards.\n")
	prompt()

	reader := bufio.NewReader(os.Stdin)
	result, err = reader.ReadBytes('\n')
	if err != nil {
		return "", err
	}
	result = bytes.Replace(result, []byte("\n"), []byte{}, -1)
	result = bytes.Replace(result, []byte("\r"), []byte{}, -1)

	return string(result), nil
}

func (this *Bitwarden) Sync() error {
	_, err := this.Execute(nil, "sync")
	if err != nil {
		return err
	}
	return nil
}

func (this *Bitwarden) FindItems(q ItemsQuery) (Items, error) {
	args := []string{"list", "items"}

	var atLeastOneLimitationProvided bool

	if v := q.Search; v != "" {
		args = append(args, "--search", v)
		atLeastOneLimitationProvided = true
	}
	if v := q.OrganizationId; v != "" {
		args = append(args, "--organizationid", v)
		atLeastOneLimitationProvided = true
	}
	if v := q.CollectionId; v != "" {
		args = append(args, "--collectionid", v)
		atLeastOneLimitationProvided = true
	}
	if v := q.FolderId; v != "" {
		args = append(args, "--folderid", v)
		atLeastOneLimitationProvided = true
	}

	if v := q.OnTooBroadQuery; v != nil && !atLeastOneLimitationProvided {
		v()
	}

	args = append(args, "--raw")

	var items Items
	err := this.ExecuteAndUnmarshal(nil, &items, args...)
	if err != nil {
		return nil, err
	}

	for i, item := range items {
		if err := item.ResolveAttachments(q.Attachments, this); err != nil {
			return nil, err
		}
		items[i] = item
	}

	return items, nil
}

func (this *Bitwarden) FindItem(q ItemQuery) (*Item, error) {
	args := []string{"list", "items"}

	var atLeastOneLimitationProvided bool

	if v := q.Name; v != "" {
		args = append(args, "--search", v)
		atLeastOneLimitationProvided = true
	} else {
		return nil, fmt.Errorf("no name in item query provided")
	}

	if v := q.OrganizationId; v != "" {
		args = append(args, "--organizationid", v)
		atLeastOneLimitationProvided = true
	}
	if v := q.CollectionId; v != "" {
		args = append(args, "--collectionid", v)
		atLeastOneLimitationProvided = true
	}
	if v := q.FolderId; v != "" {
		args = append(args, "--folderid", v)
		atLeastOneLimitationProvided = true
	}

	if v := q.OnTooBroadQuery; v != nil && !atLeastOneLimitationProvided {
		v()
	}

	args = append(args, "--raw")

	var items Items
	err := this.ExecuteAndUnmarshal(nil, &items, args...)
	if err != nil {
		return nil, err
	}

	var match *Item
	for _, item := range items {
		if item.Name == q.Name {
			if match != nil {
				return nil, ErrItemNotUnique
			}
			v := item
			match = &v
		}
	}

	if match == nil {
		return nil, ErrNoSuchItem
	}

	if err := match.ResolveAttachments(q.Attachments, this); err != nil {
		return nil, err
	}

	return match, nil
}

func (this *Bitwarden) GetItem(id string) (*Item, error) {
	var item Item
	err := this.ExecuteAndUnmarshal(nil, &item, "get", "item", id)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (this *Bitwarden) CreateAttachment(of Item, attachmentName string, attachment Attachment) (gErr error) {
	defer func() {
		if gErr != nil {
			gErr = fmt.Errorf("cannot create attachment '%s' for item %s (%s): %w", attachmentName, of.Name, of.Id, gErr)
		}
	}()

	if this.legacyAttachment {
		file, err := attachment.ToTempFile(attachmentName)
		if err != nil {
			return err
		}
		defer func() {
			if err := file.Close(); err != nil && gErr == nil {
				gErr = err
			}
		}()
		if _, err := this.Execute(nil, "create", "attachment", "--itemid", of.Id, "--file", file.Name); err != nil {
			return err
		}

	} else {
		if _, err := this.Execute(func(cmd *exec.Cmd) {
			cmd.Stdin = attachment.ToReader()
		}, "create", "attachment", "--itemid", of.Id, "--file", attachmentName, "--stdin"); err != nil {
			return err
		}
	}
	log.With("itemName", of.Name).
		With("itemId", of.Id).
		With("attachmentName", attachmentName).
		Debug("Attachment created.")

	return nil
}

func (this *Bitwarden) DeleteAttachment(of Item, attachment ItemAttachmentReference) (gErr error) {
	defer func() {
		if gErr != nil {
			gErr = fmt.Errorf("cannot delete attachment '%s' (%s) for item %s (%s): %w", attachment.FileName, attachment.Id, of.Name, of.Id, gErr)
		}
	}()
	v, err := this.Execute(nil, "delete", "attachment", "--itemid", of.Id, attachment.Id)
	if err != nil {
		return err
	}
	log.With("itemName", of.Name).
		With("itemId", of.Id).
		With("attachmentName", attachment.FileName).
		With("attachmentId", attachment.Id).
		With("response", string(v)).
		Debug("Attachment deleted.")

	return nil
}

func (this *Bitwarden) GetAttachments(of Item, by ItemAttachmentQueries) (ItemAttachments, error) {
	result := ItemAttachments{}
	for _, ref := range of.AttachmentReferences {
		for _, q := range by {
			if q.FilenameMatches.MatchString(ref.FileName) {
				v, err := this.GetAttachment(of, ref.Id, q.Base64Encode)
				if err != nil {
					return nil, err
				}
				if _, alreadyExists := result[q.Name]; alreadyExists && q.Unique {
					return nil, fmt.Errorf("%v: more then one attachment matching %v but it should be unique", q.Name, q.FilenameMatches)
				}
				result[q.Name] = v
			}
		}
	}
	return result, nil
}

func (this *Bitwarden) GetAttachment(of Item, attachmentId string, base64encoded bool) (string, error) {
	v, err := this.Execute(nil, "get", "attachment", attachmentId, "--itemid", of.Id, "--raw")
	if err != nil {
		return "", fmt.Errorf("cannot get attachment %s of item %v (%v)", attachmentId, of.Name, of.Id)
	}
	if base64encoded {
		return base64.StdEncoding.EncodeToString(v), nil
	}
	return string(v), nil
}

type CommandCustomizer func(*exec.Cmd)

func (this *Bitwarden) ExecuteAndUnmarshal(customizer CommandCustomizer, to interface{}, args ...string) error {
	stdout, err := this.Execute(customizer, args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(stdout, to); err != nil {
		return this.Errorf(args, "%w", err)
	}

	return nil
}

func (this *Bitwarden) Execute(customizer CommandCustomizer, args ...string) ([]byte, error) {
	stdout, stderr, err := this.ExecuteDirect(customizer, args...)
	if eErr, ok := err.(*exec.ExitError); ok {
		return nil, fmt.Errorf("%w: %s", eErr, stderr)
	} else if err != nil {
		return nil, fmt.Errorf("%w: %s", err, stderr)
	}

	if strings.HasPrefix(stderr, "mac failed.\n") {
		return nil, this.Errorf(args, "%w", ErrWrongSession)
	}

	return stdout, nil
}

func (this *Bitwarden) ExecuteDirect(customizer CommandCustomizer, args ...string) ([]byte, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(this.executable[0], append(this.executable[1:], args...)...)
	cmd.Dir = "C:\\development\\github.com\\blaubaer\\cli"
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = utils.AddEnvironment(os.Environ(), map[string]string{
		"BW_SESSION": this.session,
	})
	if customizer != nil {
		customizer(cmd)
	}

	err := cmd.Run()
	if err != nil {
		return stdout.Bytes(), stderr.String(), this.Errorf(args, "unexpected error: %w", err)
	}
	return stdout.Bytes(), stderr.String(), nil
}

func (this *Bitwarden) Errorf(args []string, msg string, msgArgs ...interface{}) error {
	targetMsg := fmt.Sprintf("%s: %s", this.FormatArgs(args), msg)
	return fmt.Errorf(targetMsg, msgArgs...)
}

func (this *Bitwarden) FormatArgs(args []string) string {
	args = append(this.executable, args...)
	bufs := make([]string, len(args))
	for i, arg := range args {
		bufs[i] = fmt.Sprintf("%q", arg)
	}
	return "[" + strings.Join(bufs, ", ") + "]"
}

func additionalExtensions() []string {
	switch runtime.GOOS {
	case "windows":
		parts := os.Getenv("PATHEXT")
		if strings.TrimSpace(parts) == "" {
			parts = ".COM;.EXE;.BAT;.CMD"
		}
		return strings.Split(parts, ";")
	default:
		return nil
	}
}

func lookupExecutable(what string) (string, error) {
	result, err := exec.LookPath(what)
	if err == nil || !os.IsNotExist(err) {
		return result, err
	}

	for _, ext := range additionalExtensions() {
		result, err := exec.LookPath(what + ext)
		if err == nil {
			return result, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}

	return "", os.ErrNotExist
}
