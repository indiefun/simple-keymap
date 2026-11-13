const http = require("node:http");
const fs = require("node:fs");
const path = require("node:path");
const root = path.resolve(__dirname, "../site");
const types = {".html":"text/html; charset=utf-8", ".css":"text/css; charset=utf-8", ".js":"text/javascript; charset=utf-8", ".txt":"text/plain; charset=utf-8", ".xml":"application/xml; charset=utf-8", ".png":"image/png", ".jpg":"image/jpeg", ".svg":"image/svg+xml", ".ico":"image/x-icon", ".zip":"application/zip"};
function handler(request, response) {
  let pathname;
  try { pathname = decodeURIComponent(new URL(request.url, "http://localhost").pathname); } catch { response.writeHead(400).end(); return; }
  const filename = path.resolve(root, "." + (pathname === "/" ? "/index.html" : pathname));
  if (!filename.startsWith(root + path.sep)) { response.writeHead(403).end(); return; }
  fs.readFile(filename, (error, bytes) => {
    if (error) { response.writeHead(404, {"Content-Type":"text/html; charset=utf-8"}); response.end(fs.readFileSync(path.join(root,"404.html"))); return; }
    response.writeHead(200, {"Content-Type":types[path.extname(filename)] || "application/octet-stream", "Cache-Control":"no-store"});
    response.end(bytes);
  });
}
module.exports = {handler, root};
if (require.main === module) {
  const port = Number(process.env.PORT || 4173);
  http.createServer(handler).listen(port, "127.0.0.1", () => console.log(`轻映网站预览：http://127.0.0.1:${port}`));
}
