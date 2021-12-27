package main

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Resp struct {
	Entries *[]Entry
	Err     error
	Stats   *Stats
	URL     string
	Offset  int
	User    string
	Guild   string
	Channel string
	Search  string
}

var funcMap = template.FuncMap{
	"add":      add,
	"subtract": subtract,
	"setQuery": setQuery,
	"intToStr": intToStr,
}

const index = `
	<!DOCTYPE HTML>
	<html>
	<head>
	<style>
	body {
		display: block;
		font-family: monospace;
		white-space: wrap;
		text-align: center;
	}
	div{
		margin: 1em 0;
		max-width: 70em;
		margin-top: 0.5em;
		margin-bottom: 0.5em;
		margin-left: auto;
		margin-right: auto;
	}
	table, th, td {
		border: 1px solid black;
		border-collapse: collapse;
	}
	th, td {
		padding: .25em;
	}
	table {
		table-layout: fixed;
		width: 100%;
	}
	th {
		text-align: left;
		word-break: break-all;
	}
	td {
		vertical-align: top;
		text-align: left;
		word-break: break-all;
	}
	#navigate {
		display: flex;
		justify-content: space-between;
	}
	input {
		max-width: 12em;
	}
	</style>
	</head>
	<body>
	<div>
	<h1>Keep</h1>
	<p>{{- .Err -}}</p>
	<p>
		<b>{{- .Stats.URLs -}}</b> URLs,
		<b>{{- .Stats.Users -}}</b> users,
		<b>{{- .Stats.Guilds -}}</b> guilds,
		<b>{{- .Stats.Channels -}}</b> channels
	</p>
	<div style="padding-top:5px; padding-bottom:5px;">
	<form action="" method="get">
		<input type="text" id="user" name="user" placeholder="User ID">
		<input type="text" id="guild" name="guild" placeholder="Guild ID">
		<input type="text" id="channel" name="channel" placeholder="Channel ID">
		<input type="text" id="search" name="search" placeholder="URL Search">
		<input type="submit" value="Filter">
	</form>
	</div>
	<p>
		{{- if or (ne .User "") (ne .Guild "") (ne .Channel "") (ne .Search "" ) -}}
		Entries filtered by:
		{{- end -}}
		{{- if ne .User "" }} <b>User</b> ({{ .User -}}){{- end -}}
		{{- if ne .Guild "" }} <b>Guild</b> ({{ .Guild -}}){{- end -}}
		{{- if ne .Channel "" }} <b>Channel</b> ({{ .Channel -}}){{- end -}}
		{{- if ne .Search "" }} <b>URL</b> ({{ .Search -}}){{- end -}}
	</p>
	{{- if gt (len .Entries) 0 -}}
	<div id="navigate">
	{{- if gt .Offset 0 -}}
	<a href="{{ setQuery .URL "offset" (intToStr (subtract .Offset 100)) }}">Previous</a>
	{{- end -}}
	<a href="./">Home</a>
	{{- if ge (len .Entries) 100 -}}
	<a href="{{ setQuery .URL "offset" (intToStr (add .Offset 100)) }}">Next</a>
	{{- end -}}
	</div>
	<table>
    <colgroup>
		<col span="1" style="width: 7%;">
		<col span="1" style="width: 5%;">
		<col span="1" style="width: 87%;">
    </colgroup>
	<tr>
	<th>ID</th>
	<th>HTTP</th>
	<th>URL</th>
	</tr>
	{{- range $e := .Entries -}}
	<tr>
	<td>{{- $e.ID -}}</td>
	<td>{{- if eq $e.Status 0 -}}PEND{{- else -}}{{ $e.Status }}{{- end -}}</td>
	<td><a href="{{ $e.Message.URL }}">{{ $e.Message.URL }}</a></td>
	</tr>
	{{- end -}}
	</table>
	</div>
	{{- else -}}
	<p>No results to display</p>
	<p><a href="./">Home</a></p>
	{{- end -}}
	{{- if gt (len .Entries) 0 -}}
	<div id="navigate">
	{{- if gt .Offset 0 -}}
	<a href="{{ setQuery .URL "offset" (intToStr (subtract .Offset 100)) }}">Previous</a>
	{{- end -}}
	<a href="./">Home</a>
	{{- if ge (len .Entries) 100 -}}
	<a href="{{ setQuery .URL "offset" (intToStr (add .Offset 100)) }}">Next</a>
	{{- end -}}
	</div>
	{{- end -}}
	</body>
	</html>
	`

var indexTmp = template.Must(template.New("").Funcs(funcMap).Parse(index))

func add(a int, b int) int {

	return a + b
}

func subtract(a int, b int) int {

	return a - b
}

func intToStr(a int) string {

	return strconv.Itoa(a)
}

func setQuery(urlStr string, query string, value string) string {

	u, _ := url.Parse(urlStr)
	q := u.Query()
	q.Set(query, value)
	u.RawQuery = q.Encode()
	return strings.TrimLeft(u.String(), "/")
}

func (db *SqliteDB) IndexHandler(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != "/" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	resp := Resp{}
	resp.Stats, resp.Err = db.Stats()
	if resp.Err != nil {
		log.Println(resp.Err)
		indexTmp.Execute(w, &resp)
		return
	}

	resp.URL = r.URL.String()
	query := r.URL.Query()

	var err error
	resp.Offset, err = strconv.Atoi(query.Get("offset"))
	if err != nil {
		resp.Offset = 0
	}
	resp.User = query.Get("user")
	resp.Guild = query.Get("guild")
	resp.Channel = query.Get("channel")
	resp.Search = query.Get("search")

	resp.Entries, resp.Err = db.ListEntries(100, resp.Offset, resp.User,
		resp.Guild, resp.Channel, resp.Search)
	if resp.Err != nil {
		log.Println(resp.Err)
	}
	indexTmp.Execute(w, &resp)
}
