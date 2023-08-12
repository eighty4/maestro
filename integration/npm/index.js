const http = require('node:http')

const server = http.createServer((req, res) => res.end())

server.listen(8000)
