# POSIX awk support for the relay's data boundaries. No shell commands run here.
function trim(s) { sub(/^[[:space:]]+/, "", s); sub(/[[:space:]]+$/, "", s); return s }
function fail(s) { print "pstack-relay: " s > "/dev/stderr"; failed=1; exit 1 }
function quote(s,    i,c,r,n) {
 r="\""
 for(i=1;i<=length(s);i++) { c=substr(s,i,1); n=index(controls,c)
  if(c=="\"" || c=="\\") r=r "\\" c
  else if(n) r=r sprintf("\\u%04x",n)
  else r=r c
 }
 return r "\""
}
function effort(e) { return e ~ /^(low|medium|high|xhigh|max|session)$/ }
function harness(h) { return h ~ /^(claude|codex|pi)$/ }
function keynorm(s,    a,n,i,r) { n=split(s,a,","); r=trim(a[1]); for(i=2;i<=n;i++) r=r ", " trim(a[i]); return r }
function parse(line,ln,    p,k,v,a,b,n,j,c,h,m,e,nf,parts) {
 if(length(line)>=1048576) fail("line " ln ": bufio.Scanner: token too long")
 line=trim(line); if(line=="" || substr(line,1,1)=="#") return
 p=index(line,":"); k=trim(substr(line,1,p-1)); v=trim(substr(line,p+1))
 if(!p || k=="" || v=="") fail("line " ln ": expected role: value")
 if(k=="default effort") { if(!effort(v)) fail("line " ln ": unknown effort " v); defaultEffort=v; return }
 if(k=="session hook") { if(v!="on" && v!="off") fail("line " ln ": session hook must be on or off"); hook=(v=="on"); return }
 n=split(k,a,","); for(j=1;j<=n;j++) if(trim(a[j])=="") fail("line " ln ": invalid role " k)
 k=keynorm(k); if(!(k in counts)) order[++roles]=k
 n=split(v,parts,","); counts[k]=n
 for(j=1;j<=n;j++) {
  nf=split(trim(parts[j]),b,/[[:space:]]+/)
  if(trim(parts[j])=="" || nf>2) fail("line " ln ": expected [harness:]model [@effort]")
  h=""; m=b[1]; e=""; p=index(m,":")
  if(p) { h=substr(m,1,p-1); m=substr(m,p+1); if(!harness(h)) fail("line " ln ": unknown harness " h); if(m=="" || index(m,":")) fail("line " ln ": invalid model " m) }
  if(nf==2) { e=substr(b[2],2); if(substr(b[2],1,1)!="@" || !effort(e)) fail("line " ln ": unknown effort " b[2]) }
  hs[k,j]=h; ms[k,j]=m; es[k,j]=e
 }
}
function sheetread(path,    s,n,rc) {
 while((rc=(getline s < path))>0) parse(s,++n)
 close(path); if(rc<0) fail("cannot read sheet " path)
}
function format(    r,i,j,k,v,t) {
 for(i=1;i<=roles;i++) { k=order[i]; t=""; for(j=1;j<=counts[k];j++) { v=ms[k,j]; if(hs[k,j]!="") v=hs[k,j] ":" v; if(es[k,j]!="") v=v " @" es[k,j]; t=t (j>1?", ":"") v }; r=r k ": " t "\n" }
 if(defaultEffort!="") r=r "default effort: " defaultEffort "\n"
 return r "session hook: " (hook?"on":"off") "\n"
}
function sheetjson(    r,i,j,k,t,o) {
 r="{\"roles\":{"; o="["
 for(i=1;i<=roles;i++) { k=order[i]; t="["; for(j=1;j<=counts[k];j++) t=t (j>1?",":"") "{\"harness\":" quote(hs[k,j]) ",\"model\":" quote(ms[k,j]) ",\"effort\":" quote(es[k,j]) "}"
 r=r (i>1?",":"") quote(k) ":" t "]"; o=o (i>1?",":"") quote(k) }
 return r "},\"order\":" (roles?o "]":"null") ",\"default_effort\":" quote(defaultEffort) ",\"session_hook\":" (hook?"true":"false") "}"
}
function resolve(    role,k,i,j,a,n,m,h,e,mode,note,cmd,p,own,backend,branch,out) {
 role=ENVIRON["role"]; k=keynorm(role); p=ENVIRON["parent"]; backend=ENVIRON["backend"]; own=ENVIRON["own"]; branch=ENVIRON["branch"]
 if(p!="" && !harness(p)) fail("unknown parent harness " quote(p))
 if(backend!="" && backend !~ /^(orca|rex|hc|none)$/) fail("unknown backend " quote(backend))
 if(!(k in counts)) { k=""; for(i=1;i<=roles && k=="";i++) { n=split(order[i],a,","); for(j=1;j<=n;j++) if(trim(a[j])==keynorm(role)) { k=order[i]; break } } }
 if(k=="") { print "[{\"role\":" quote(role) ",\"seat\":0,\"mode\":\"native\",\"harness\":" quote(p) ",\"model\":\"\",\"effort\":\"\",\"note\":\"role not in sheet; use the skill\047s default model\"}]"; return }
 out="["
 for(i=1;i<=counts[k];i++) {
  m=ms[k,i]; if(m=="auto" || m=="inherit-parent") m=""
  e=es[k,i]; if(e=="") e=defaultEffort; if(e=="session") e=""
  h=p; mode="native"; note=""; cmd=""
  if(p=="") note="parent harness not detected"
  else if(hs[k,i]=="" || hs[k,i]==p || m=="") { }
  else if(backend=="orca" && hs[k,i]!="pi") {
   h=hs[k,i]; mode="orca"; cmd="[\"orca\",\"orchestration\",\"worker-start\",\"--agent\"," quote(h) ",\"--model\"," quote(m)
   if(e!="") cmd=cmd ",\"--effort\"," quote(e)
   if(own=="true") { if(branch=="") fail("own worktree requires a current branch for --base-branch"); cmd=cmd ",\"--worktree\",\"new-child\",\"--base-branch\"," quote(branch) }
   else cmd=cmd ",\"--worktree\",\"current\""
   cmd=cmd ",\"--spec-file\",\"{brief}\"]"
  } else if(backend=="rex" || backend=="hc") {
   mode=backend; h=hs[k,i]; cmd="[" quote(ENVIRON["self"]) "," quote(backend) ",\"start\",\"--harness\"," quote(h) ",\"--model\"," quote(m)
   if(e!="") cmd=cmd ",\"--effort\"," quote(e)
   if(own=="true") cmd=cmd ",\"--own-worktree\""
   cmd=cmd ",\"--brief\",\"{brief}\"]"
  } else {
   mode="fallback"; m=""; note="configured seat " hs[k,i] ":" ms[k,i] ": " ((backend=="orca" && hs[k,i]=="pi")?"Orca has no pi agent":"no delegation backend detected") "; fall back to parent model"
  }
  out=out (i>1?",":"") "{\"role\":" quote(role) ",\"seat\":" i-1 ",\"mode\":" quote(mode) ",\"harness\":" quote(h) ",\"model\":" quote(m) ",\"effort\":" quote(e)
  if(cmd!="") out=out ",\"command\":" cmd
  if(note!="") out=out ",\"note\":" quote(note)
  out=out "}"
 }
 print out "]"
}
# JSON recursive descent: values are indexed by node id, never evaluated as code.
function ws() { while(substr(src,pos,1) ~ /[ \t\r\n]/ && pos<=length(src)) pos++ }
function utf8(n) {
 if(n<128) return sprintf("%c",n)
 if(n<2048) return sprintf("%c%c",192+int(n/64),128+n%64)
 if(n<65536) return sprintf("%c%c%c",224+int(n/4096),128+int(n/64)%64,128+n%64)
 return sprintf("%c%c%c%c",240+int(n/262144),128+int(n/4096)%64,128+int(n/64)%64,128+n%64)
}
function hex4(    i,c,n) { n=0; for(i=0;i<4;i++) { c=index("0123456789abcdef",tolower(substr(src,pos++,1)))-1; if(c<0) fail("invalid JSON unicode escape"); n=n*16+c }; return n }
function stringval(    r,c,n,lo) {
 if(substr(src,pos++,1)!="\"") fail("expected JSON string")
 while(pos<=length(src)) {
  c=substr(src,pos++,1); if(c=="\"") return r
  if(c=="\\") { c=substr(src,pos++,1)
   if(c=="u") { n=hex4(); if(n>=55296 && n<=56319) { if(substr(src,pos,2)!="\\u") fail("invalid JSON surrogate"); pos+=2; lo=hex4(); if(lo<56320 || lo>57343) fail("invalid JSON surrogate"); n=65536+(n-55296)*1024+lo-56320 }; if(n==0) fail("NUL is not supported in command arguments"); r=r utf8(n) }
   else if(c=="n") r=r "\n"; else if(c=="r") r=r "\r"; else if(c=="t") r=r "\t"; else if(c=="b") r=r sprintf("%c",8); else if(c=="f") r=r sprintf("%c",12)
   else if(c=="\\" || c=="/" || c=="\"") r=r c; else fail("invalid JSON escape")
  } else { if(index(controls,c)) fail("invalid JSON control character"); r=r c }
 }
 fail("unterminated JSON string")
}
function value(    id,c,k,v,n,start) {
 ws(); id=++nodes; c=substr(src,pos,1); start=pos
 if(c=="{") { typ[id]="object"; pos++; ws(); if(substr(src,pos,1)=="}") pos++
  else while(1) { ws(); k=stringval(); ws(); if(substr(src,pos++,1)!=":") fail("invalid JSON object"); v=value(); child[id,k]=v; ws(); c=substr(src,pos++,1); if(c=="}") break; if(c!=",") fail("invalid JSON object") }
 } else if(c=="[") { typ[id]="array"; pos++; ws(); if(substr(src,pos,1)=="]") pos++
  else while(1) { child[id,++n]=value(); ws(); c=substr(src,pos++,1); if(c=="]") break; if(c!=",") fail("invalid JSON array") }; size[id]=n
 } else if(c=="\"") { typ[id]="string"; val[id]=stringval() }
 else { while(pos<=length(src) && substr(src,pos,1)!~/[ \t\r\n,}\]]/) pos++; v=substr(src,start,pos-start)
  if(v=="true" || v=="false" || v=="null") typ[id]=v
  else if(v ~ /^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$/) { typ[id]="number"; val[id]=v }
  else fail("invalid JSON value")
 }
 raw[id]=substr(src,start,pos-start); return id
}
function json(s,    id) { src=s; pos=1; id=value(); ws(); if(pos<=length(src)) fail("invalid JSON trailing data"); return id }
function chomp(s) { sub(/\n$/, "", s); return s }
function readall(path,    r,s,rc) { while((rc=(getline s < path))>0) r=r s "\n"; close(path); if(rc<0) fail("cannot read " path); return r }
function arrmodels(h,    r,i) { r="["; for(i=1;i<=nm[h];i++) r=r (i>1?",":"") quote(models[h,i]); return r "]" }
function catalogs(    r,i,h) { r="["; for(i=1;i<=3;i++) { h=horder[i]; r=r (i>1?",":"") "{\"harness\":" quote(h) ",\"installed\":" (installed[h]?"true":"false") ",\"models\":" arrmodels(h); if(notes[h]!="") r=r ",\"note\":" quote(notes[h]); r=r "}" }; return r "]" }
function loadcatalogs(    root,i,j,h,id,ar,model,first,seen,a,n,f,line) {
 root=ENVIRON["scratch"]
 for(i=1;i<=3;i++) {
  h=horder[i]; installed[h]=(trim(readall(root "/" h ".installed"))=="yes")
  if(!installed[h]) { notes[h]="not available: " chomp(readall(root "/" h ".error")); continue }
  if(h=="claude") { nm[h]=4; models[h,1]="opus"; models[h,2]="sonnet"; models[h,3]="haiku"; models[h,4]="fable"; notes[h]="aliases; Claude has no model list command"; continue }
  if(trim(readall(root "/" h ".error"))!="") { notes[h]="model discovery failed: " chomp(readall(root "/" h ".error")); continue }
  if(h=="codex") {
   id=json(readall(root "/" h ".models")); ar=child[id,"models"]
   if(ar && typ[ar]!="array" && typ[ar]!="null") fail("models must be an array")
   for(j=1;j<=size[ar];j++) { model=child[child[ar,j],"slug"]; if(model && typ[model]!="string" && typ[model]!="null") fail("slug must be a string"); if(val[model]!="") models[h,++nm[h]]=val[model] }
  } else {
   n=split(readall(root "/" h ".models"),a,"\n")
   for(j=1;j<=n;j++) { line=trim(a[j]); f=split(line,seen,/[[:space:]]+/); if(line=="" || seen[1]~/^Warning:/ || tolower(seen[1])=="provider") continue
    model=""; if(index(seen[1],"/")) model=seen[1]; else if(f>=2 && substr(seen[1],1,1)!="-") model=seen[1] "/" seen[2]
    if(model!="" && !((h SUBSEP model) in unique)) { unique[h,model]=1; models[h,++nm[h]]=model }
   }
  }
  if(!nm[h]) notes[h]="model discovery returned no models"
 }
}
function bracket(s,    i,c,q,d) { for(i=1;i<=length(s);i++) { c=substr(s,i,1); if(q!="") { if(c==q) q="" } else if(c=="\"" || c=="\047") q=c; else if(c=="#") break; else if(c=="[") d++; else if(c=="]") d-- }; return d }
function toml(path,    a,n,i,line,depth,triple,p,k,v,q,r,c,j) {
 n=split(readall(path),a,"\n")
 for(i=1;i<=n;i++) { line=trim(a[i]); if(triple!="") { if(index(line,triple)) triple=""; continue }; if(depth>0) { depth+=bracket(line); continue }
  if(substr(line,1,1)=="[") return ""
  p=index(line,"="); if(!p) continue; k=trim(substr(line,1,p-1)); gsub(/^[\047"]+|[\047"]+$/,"",k); v=trim(substr(line,p+1))
  if((substr(v,1,3)=="\"\"\"" || substr(v,1,3)=="\047\047\047") && !index(substr(v,4),substr(v,1,3))) { triple=substr(v,1,3); continue }
  if(substr(v,1,1)=="[") { depth=bracket(v); continue }
  if(k=="model") { q=substr(v,1,1); if(q!="\"" && q!="\047") return ""; r=""; for(j=2;j<=length(v);j++) { c=substr(v,j,1); if(c==q) return r; if(c=="\\" && q=="\"" && j<length(v)) c=substr(v,++j,1); r=r c }; return "" }
 }
 return ""
}
function prefer(    m,j,k) { if(ENVIRON["config"]=="") return; m=toml(ENVIRON["config"]); for(j=2;j<=nm["codex"];j++) if(models["codex",j]==m) { for(k=j;k>1;k--) models["codex",k]=models["codex",k-1]; models["codex",1]=m; break } }
function starter(    template,a,n,i,on,found,k,j,parts,np,family,e,h,m,b) {
 template=readall(ENVIRON["template"]); n=split(template,a,"```")
 for(i=2;i<=n;i+=2) if(index(a[i],"feature, refactoring:")) { template=a[i]; found=1; break }
 if(!found) fail("upstream setup template has no role block")
 n=split(template,a,"\n"); found=0
 for(i=1;i<=n;i++) { if(trim(a[i])~/^feature, refactoring:/) found=1; if(found) parse(a[i],++on) }
 if(!nm["claude"] && !nm["codex"] && !nm["pi"]) fail("no installed harness offers a model")
 for(i=1;i<=roles;i++) { k=order[i]; for(j=1;j<=counts[k];j++) {
  np=split(ms[k,j],parts,"-"); family=(parts[1]=="claude" && np>1?parts[2]:""); e="high"
  for(b=np;b>=1;b--) if(effort(parts[b]) && parts[b]!="session") { e=parts[b]; break }
  if(family!="" && nm["claude"]) { h="claude"; m=family } else if(nm["codex"]) { h="codex"; m=models[h,1] } else if(nm["claude"]) { h="claude"; m=models[h,1] } else { h="pi"; m=models[h,1] }
  hs[k,j]=h; ms[k,j]=m; es[k,j]=e
 } }
 defaultEffort="session"
}
BEGIN {
 for(ci=1;ci<32;ci++) controls=controls sprintf("%c",ci)
 hook=1; horder[1]="codex"; horder[2]="claude"; horder[3]="pi"; action=ENVIRON["action"]
 if(action=="quote") { printf "%s",quote(ENVIRON["text"]); exit }
 if(action=="json") { id=json(readall(ENVIRON["input"])); n=split(ENVIRON["field"],a,"."); for(i=1;i<=n;i++) id=child[id,a[i]]; if(ENVIRON["raw"]=="true") print raw[id]; else if(typ[id]=="string" || typ[id]=="number") printf "%s", val[id]; else if(typ[id]!="" && typ[id]!="null") fail("unexpected JSON field type"); exit }
 if(action=="rexlaunchvalidate" || action=="rexprocessvalidate") {
  id=json(readall(ENVIRON["input"])); if(typ[id]!="object" && typ[id]!="null") fail("expected Rex object")
  if(action=="rexlaunchvalidate") {
   b=child[id,"block_id"]; if(b && typ[b]!="string" && typ[b]!="null") fail("expected block id string")
   b=child[id,"block_ids"]; if(b && typ[b]!="array" && typ[b]!="null") fail("expected block ids array")
   for(i=1;i<=size[b];i++) if(typ[child[b,i]]!="string" && typ[child[b,i]]!="null") fail("expected block id string")
  } else {
   b=child[id,"last_exit"]; if(b && typ[b]!="object" && typ[b]!="null") fail("expected Rex last_exit object")
   b=child[b,"exit_code"]; if(b && typ[b]!="null" && (typ[b]!="number" || val[b]!~/^-?[0-9]+$/)) fail("expected integer exit code")
  }; exit
 }
 if(action=="codexvalidate") {
  id=json(readall(ENVIRON["input"])); if(typ[id]!="object" && typ[id]!="null") fail("expected models object")
  ar=child[id,"models"]; if(ar && typ[ar]!="array" && typ[ar]!="null") fail("expected models array")
  for(i=1;i<=size[ar];i++) { item=child[ar,i]; if(typ[item]!="object" && typ[item]!="null") fail("expected model object"); slug=child[item,"slug"]; if(slug && typ[slug]!="string" && typ[slug]!="null") fail("expected slug string") }; exit
 }
 if(action=="validate") { json(readall(ENVIRON["input"])); exit }
 if(action=="catalogs" || action=="starter" || action=="setup" || action=="doctor") loadcatalogs()
 if(action=="doctor") { for(i=1;i<=3;i++) { h=horder[i]; print "{\"name\":" quote(h) ",\"ok\":" ((!installed[h] || nm[h]>0)?"true":"false") ",\"detail\":" quote("installed=" (installed[h]?"true":"false") " models=" (nm[h]+0) " " notes[h]) "}" }; exit }
 if(action=="setup") { for(i=1;i<=3;i++) { h=horder[i]; t=""; for(j=1;j<=nm[h];j++) t=t (j>1?", ":"") models[h,j]; print h ": installed=" (installed[h]?"true":"false") " models=" t " " notes[h] }; exit }
 if(action=="catalogs") { prefer(); print catalogs(); exit }
 if(action=="starter") { prefer(); starter(); printf "%s",format(); exit }
 if(action=="models" || action=="resolve") {
  if(ENVIRON["sheetfile"]!="") sheetread(ENVIRON["sheetfile"])
  if(ENVIRON["edit"]!="") { s=ENVIRON["edit"]; p=index(s,"="); k=trim(substr(s,1,p-1)); v=substr(s,p+1); if(!p || k=="" || k~/[:\r\n]/ || v~/[\r\n]/) fail("--set requires one role=value[, value]"); parse(k ": " v,1) }
  if(action=="resolve") resolve()
  else if(ENVIRON["asjson"]=="true") print "{\"path\":" quote(ENVIRON["sheet"]) ",\"sheet\":" sheetjson() ",\"text\":" quote(format()) "}"
  else printf "%s",format()
  exit
 }
}
