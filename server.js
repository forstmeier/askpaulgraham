var axios = require('axios');
var express = require('express');
var serveStatic = require('serve-static');

app = express();
app.use(serveStatic(__dirname + "/dist"));

var port = process.env.PORT || 3000;
var hostname = '127.0.0.1';

app.listen(port, hostname, () => {
	console.log(`Server running at http://${hostname}:${port}/`);
});

app.post('/question', async (req, res) => {
	let questionResponse = await axios.post(
		process.env.APG_QUESTION_URL,
		req,
	)
	res.json(questionResponse.data);
});

app.get('/summaries', async (req, res) => {
	let summariesResponse = await axios.get(
		process.env.APG_SUMMARIES_URL,
	);
	res.json(summariesResponse.data);
});