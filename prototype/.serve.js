const http=require("http"),fs=require("fs"),path=require("path");
const root=__dirname;
const mime={".html":"text/html; charset=utf-8",".css":"text/css; charset=utf-8",".js":"text/javascript; charset=utf-8",".svg":"image/svg+xml",".json":"application/json"};
http.createServer((req,res)=>{
  let p=decodeURIComponent(req.url.split("?")[0]); if(p==="/")p="/index.html";
  const f=path.join(root,p);
  fs.readFile(f,(e,d)=>{ if(e){res.writeHead(404);res.end("404");return;}
    res.writeHead(200,{"Content-Type":mime[path.extname(f)]||"application/octet-stream"});res.end(d);});
}).listen(8437,"127.0.0.1",()=>console.log("serving on 8437"));
