// Templates for webpages
package main

const ReportGroupErrorsTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Example of Table with twitter bootstrap</title>
<meta name="description" content="Creating a table with Twitter Bootstrap. Learn how to use Twitter Bootstrap toolkit to create Tables with examples.">
<link href="/twitter-bootstrap/twitter-bootstrap-v2/docs/assets/css/bootstrap.css" rel="stylesheet">
</head>
<body>
<table class="table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Status</th>
            <th>Content Length</th>
            <th>Request Duration</th>
            <th>Last Checked</th>
            <th>Description</th>
          </tr>
        </thead>
        <tbody>
        {{#StreamReport}}
          <tr>
            <td><a href="${{URI}}">${{Name}}</a></td>
            <td>${{HTTPStatus}}</td>
            <td>${{ContentLength}}</td>
            <td>${{Elapsed}}</td>
            <td>${{Started}}</td>
            <td>${{Description}}</td>
          </tr>
        {{/StreamReport}}
        </tbody>
      </table>
</body>
</html>
`
