// The MIT License (MIT)

// Copyright (c) 2017 Fabian Wenzelmann

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package gomlmmj

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

func GetLists(spool string) ([]string, error) {
	files, err := ioutil.ReadDir(spool)
	if err != nil {
		return nil, err
	}
	res := make([]string, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			res = append(res, f.Name())
		}
	}
	return res, nil
}

func parseListOutput(r io.Reader) ([]string, error) {
	// we use scanner to ignore all whitespaces, seems sometimes an
	// empty line is appended for example
	res := make([]string, 0)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		next := scanner.Text()
		next = strings.TrimSpace(next)
		if len(next) == 0 {
			continue
		}
		res = append(res, next)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

func parseCountOutput(r io.Reader) (int, error) {
	content, readErr := ioutil.ReadAll(r)
	if readErr != nil {
		return -1, readErr
	}
	str := strings.TrimSpace(string(content))
	return strconv.Atoi(str)
}

func listDir(spool, name string) string {
	return path.Join(spool, name)
}

type UserType int

const (
	Subscriber UserType = iota
	Digest
	Nomail
	Moderator
	Owner
)

func (t UserType) String() string {
	switch t {
	case Subscriber:
		return "Subscriber"
	case Digest:
		return "Digest"
	case Nomail:
		return "Nomail"
	case Moderator:
		return "Moderator"
	case Owner:
		return "Owner"
	default:
		return "Unkown user type"
	}
}

type SubRequest struct {
	Mail, Name, ModerationString, Spool                                              string
	WelcomeMail, ConfirmationMail, ForceSubscription, BeQuiet, MailAlreadySubscribed bool
	Mode                                                                             UserType
}

func NewSubRequest(mail, name string) SubRequest {
	return SubRequest{
		Mail:                  mail,
		Name:                  name,
		ModerationString:      "",
		Spool:                 "/var/spool/mlmmj",
		WelcomeMail:           true,
		ConfirmationMail:      false,
		ForceSubscription:     false,
		BeQuiet:               false,
		MailAlreadySubscribed: false,
		Mode: Subscriber,
	}
}

func (r SubRequest) GetArgs() ([]string, error) {
	subType := ""
	switch r.Mode {
	case Moderator, Owner:
		return nil, fmt.Errorf("Invalid subscription type %v for subscription", r.Mode)
	case Subscriber:
	case Digest:
		subType = "-d"
	case Nomail:
		subType = "-n"
	default:
		return nil, errors.New("Unkown subscription type")
	}
	args := []string{
		"-L", listDir(r.Spool, r.Name), "-a", r.Mail,
	}
	if subType != "" {
		args = append(args, subType)
	}
	if r.WelcomeMail {
		args = append(args, "-c")
	}
	if r.ConfirmationMail {
		args = append(args, "-C")
	}
	if r.ForceSubscription {
		args = append(args, "-f")
	}
	if r.ModerationString != "" {
		args = append(args, "-m", r.ModerationString)
	}
	if r.BeQuiet {
		args = append(args, "-q")
	}
	if !r.MailAlreadySubscribed {
		args = append(args, "-s")
	}
	return args, nil
}

type UnsubRequest struct {
	Mail, Name, Spool                                     string
	GoodBye, ConfirmationMail, BeQuiet, MailNotSubscribed bool
	// -1 for: "from all versions"
	Mode UserType
}

func NewUnsubRequest(mail, name string) UnsubRequest {
	return UnsubRequest{
		Mail:              mail,
		Name:              name,
		Spool:             "/var/spool/mlmmj",
		GoodBye:           true,
		ConfirmationMail:  false,
		BeQuiet:           false,
		MailNotSubscribed: false,
		Mode:              -1,
	}
}

func (r UnsubRequest) GetArgs() ([]string, error) {
	subType := ""
	switch r.Mode {
	case Moderator, Owner:
		return nil, fmt.Errorf("Invalid subscription type %v for unsubscription", r.Mode)
	case Subscriber:
		subType = "-N"
	case Digest:
		subType = "-d"
	case Nomail:
		subType = "-n"
	case -1:
	default:
		return nil, errors.New("Unkown subscription type")
	}
	args := []string{
		"-L", listDir(r.Spool, r.Name),
	}
	if subType != "" {
		args = append(args, subType)
	}
	if r.GoodBye {
		args = append(args, "-c")
	}
	if r.ConfirmationMail {
		args = append(args, "-C")
	}
	if r.BeQuiet {
		args = append(args, "-q")
	}
	if !r.MailNotSubscribed {
		args = append(args, "-s")
	}
	return args, nil
}

func GetMakeMLArgs(spool, name, domain, owner, lang string) ([]string, error) {
	return []string{
		"-L", name, "-d", domain, "-o", owner, "-l", lang, "-s", spool,
	}, nil
}

func GetListArgs(spool, name string, mode UserType, count bool) ([]string, error) {
	subType := ""
	switch mode {
	case -1:
	case Digest:
		subType = "-d"
	case Moderator:
		subType = "-m"
	case Nomail:
		subType = "-n"
	case Owner:
		subType = "-o"
	case Subscriber:
		subType = "-s"
	default:
		return nil, errors.New("Unkown subscription type")
	}
	args := []string{
		"-L", listDir(spool, name),
	}
	if subType != "" {
		args = append(args, subType)
	}
	if count {
		args = append(args, "-c")
	}
	return args, nil
}

type MLMMJHandler interface {
	MakeML(ctx context.Context, spool, name, domain, owner, lang string) (string, error)
	Sub(ctx context.Context, request SubRequest) (string, error)
	Unsub(ctx context.Context, request UnsubRequest) (string, error)
	List(ctx context.Context, spool, name string, mode UserType) ([]string, error)
	Count(ctx context.Context, spool, name string, mode UserType) (int, error)
}

var (
	UnwatchedList = errors.New("List was not properly added to the system.")
)

type MLMMJWrapper struct {
	lm      *ListManager
	handler MLMMJHandler
}

func NewMLMMJWrapper(spools []string, handler MLMMJHandler) (*MLMMJWrapper, error) {
	lm := NewListManager()
	if err := lm.Init(spools); err != nil {
		return nil, err
	}
	return &MLMMJWrapper{lm: lm, handler: handler}, nil
}

// TODO chown?
func (wrapper *MLMMJWrapper) MakeML(ctx context.Context, spool, name, domain, owner, lang string) (string, error) {
	// first try to create the list
	output, err := wrapper.handler.MakeML(ctx, spool, name, domain, owner, lang)
	if err != nil {
		return output, err
	}
	// creation successful, add to the manager
	wrapper.lm.AddList(listDir(spool, name))
	return output, err
}

func (wrapper *MLMMJWrapper) Sub(ctx context.Context, r SubRequest) (string, error) {
	// log this list for writing
	hasList, lock := wrapper.lm.WriteList(listDir(r.Spool, r.Name))
	defer lock()
	if !hasList {
		return "", UnwatchedList
	}
	// subscribe
	return wrapper.handler.Sub(ctx, r)
}

func (wrapper *MLMMJWrapper) Unsub(ctx context.Context, r UnsubRequest) (string, error) {
	// log list for writing
	hasList, lock := wrapper.lm.WriteList(listDir(r.Spool, r.Name))
	defer lock()
	if !hasList {
		return "", UnwatchedList
	}
	// unsub
	return wrapper.handler.Unsub(ctx, r)
}

func (wrapper *MLMMJWrapper) List(ctx context.Context, spool, name string, mode UserType) ([]string, error) {
	// lock list for reading
	hasList, lock := wrapper.lm.ReadList(listDir(spool, name))
	defer lock()
	if !hasList {
		return nil, UnwatchedList
	}
	return wrapper.handler.List(ctx, spool, name, mode)
}

func (wrapper *MLMMJWrapper) ListAllMembers(ctx context.Context, spool, name string) (subscribers, digest, nomail []string, err error) {
	// lock list for reading
	hasList, lock := wrapper.lm.ReadList(listDir(spool, name))
	defer lock()
	if !hasList {
		err = UnwatchedList
		return
	}
	// we're reading from different files, so reading them concurrently should
	// be fine
	// we write all errors to the result channel
	ch := make(chan error, 3)
	go func() {
		sub, nextErr := wrapper.handler.List(ctx, spool, name, Subscriber)
		subscribers = sub
		ch <- nextErr
	}()
	go func() {
		dg, nextErr := wrapper.handler.List(ctx, spool, name, Digest)
		digest = dg
		ch <- nextErr
	}()
	go func() {
		nm, nextErr := wrapper.handler.List(ctx, spool, name, Nomail)
		nomail = nm
		ch <- nextErr
	}()
	for i := 0; i < 3; i++ {
		nextErr := <-ch
		if err == nil {
			err = nextErr
		}
	}
	return
}

func (wrapper *MLMMJWrapper) ListAllControllers(ctx context.Context, spool, name string) (owners, moderators []string, err error) {
	// lock list for reading
	hasList, lock := wrapper.lm.ReadList(listDir(spool, name))
	defer lock()
	if !hasList {
		err = UnwatchedList
		return
	}
	// again read concurrently
	ch := make(chan error, 2)
	go func() {
		o, nextErr := wrapper.handler.List(ctx, spool, name, Owner)
		owners = o
		ch <- nextErr
	}()
	go func() {
		m, nextErr := wrapper.handler.List(ctx, spool, name, Moderator)
		moderators = m
		ch <- nextErr
	}()
	for i := 0; i < 2; i++ {
		nextErr := <-ch
		if err == nil {
			err = nextErr
		}
	}
	return
}

func (wrapper *MLMMJWrapper) Count(ctx context.Context, spool, name string, mode UserType) (int, error) {
	hasList, lock := wrapper.lm.ReadList(listDir(spool, name))
	defer lock()
	if !hasList {
		return -1, UnwatchedList
	}
	return wrapper.handler.Count(ctx, spool, name, mode)
}

type DockerHandler struct {
	URL     string
	Client  *http.Client
	Timeout time.Duration
}

func NewDockerHandler(url string) *DockerHandler {
	return &DockerHandler{URL: url,
		Client:  http.DefaultClient,
		Timeout: 10 * time.Second,
	}
}

func (handler *DockerHandler) post(ctx context.Context, cmd string, args []string) (string, error) {
	postArgs := map[string]interface{}{
		"mlmmj-command": cmd,
		"args":          args,
	}
	argsJSON, jsonErr := json.Marshal(postArgs)
	if jsonErr != nil {
		return "", jsonErr
	}
	req, reqErr := http.NewRequest("POST", handler.URL, strings.NewReader(string(argsJSON)))
	if reqErr != nil {
		return "", reqErr
	}
	req = req.WithContext(ctx)
	resp, doErr := handler.Client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if doErr != nil {
		return "", doErr
	}
	var respContent struct {
		ReturnCode int
		Output     string
	}
	if err := json.NewDecoder(resp.Body).Decode(&respContent); err != nil {
		return "", err
	}
	if respContent.ReturnCode != 0 {
		return respContent.Output, errors.New(respContent.Output)
	}
	return respContent.Output, nil
}

func (handler *DockerHandler) MakeML(ctx context.Context, spool, name, domain, owner, lang string) (string, error) {
	args, argsErr := GetMakeMLArgs(spool, name, domain, owner, lang)
	if argsErr != nil {
		return "", argsErr
	}
	return handler.post(ctx, "mlmmj-make-ml", args)
}

func (handler *DockerHandler) Sub(ctx context.Context, r SubRequest) (string, error) {
	args, argsErr := r.GetArgs()
	if argsErr != nil {
		return "", argsErr
	}
	return handler.post(ctx, "mlmmj-sub", args)
}

func (handler *DockerHandler) Unsub(ctx context.Context, r UnsubRequest) (string, error) {
	args, argsErr := r.GetArgs()
	if argsErr != nil {
		return "", argsErr
	}
	return handler.post(ctx, "mlmmj-unsub", args)
}

func (handler *DockerHandler) List(ctx context.Context, spool, name string, mode UserType) ([]string, error) {
	args, argsErr := GetListArgs(spool, name, mode, false)
	if argsErr != nil {
		return nil, argsErr
	}
	out, err := handler.post(ctx, "mlmmj-list", args)
	if err != nil {
		return nil, err
	}
	return parseListOutput(strings.NewReader(out))
}

func (handler *DockerHandler) Count(ctx context.Context, spool, name string, mode UserType) (int, error) {
	args, argsErr := GetListArgs(spool, name, mode, true)
	if argsErr != nil {
		return -1, argsErr
	}
	out, err := handler.post(ctx, "mlmmj-list", args)
	if err != nil {
		return -1, err
	}
	return parseCountOutput(strings.NewReader(out))
}
