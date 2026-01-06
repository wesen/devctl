package logjs

const helpersJS = `
(function(){
  function parseJSON(line) {
    try { return JSON.parse(line); } catch (e) { return null; }
  }

  // Extremely small logfmt-ish parser: key=value pairs separated by spaces.
  // Supports quoted values with \" escapes.
  function parseLogfmt(line) {
    if (typeof line !== "string") return null;
    const out = {};
    let i = 0;
    const n = line.length;

    function skipSpaces() { while (i < n && line[i] === " ") i++; }
    function parseKey() {
      const start = i;
      while (i < n && line[i] !== "=" && line[i] !== " ") i++;
      if (i === start) return null;
      return line.slice(start, i);
    }
    function parseValue() {
      if (i >= n) return "";
      if (line[i] === "\"") {
        i++;
        let s = "";
        while (i < n) {
          const ch = line[i];
          if (ch === "\"") { i++; return s; }
          if (ch === "\\\\") {
            i++;
            if (i < n) { s += line[i]; i++; continue; }
            return s;
          }
          s += ch;
          i++;
        }
        return s;
      }
      const start = i;
      while (i < n && line[i] !== " ") i++;
      return line.slice(start, i);
    }

    while (i < n) {
      skipSpaces();
      if (i >= n) break;
      const key = parseKey();
      if (!key) break;
      if (i >= n || line[i] !== "=") { out[key] = true; continue; }
      i++; // skip '='
      const value = parseValue();
      out[key] = value;
      skipSpaces();
    }
    return out;
  }

  function namedCapture(line, re) {
    if (typeof line !== "string") return null;
    if (!(re instanceof RegExp)) return null;
    const m = re.exec(line);
    if (!m) return null;
    if (!m.groups) return null;
    const out = {};
    for (const k of Object.keys(m.groups)) out[k] = m.groups[k];
    return out;
  }

  function extract(line, re, group) {
    if (typeof line !== "string") return null;
    if (!(re instanceof RegExp)) return null;
    const m = re.exec(line);
    if (!m) return null;
    const idx = (typeof group === "number") ? group : 1;
    const v = m[idx];
    return (typeof v === "string") ? v : null;
  }

  function field(obj, path) {
    if (!obj || typeof path !== "string" || path === "") return null;
    const parts = path.split(".");
    let cur = obj;
    for (const p of parts) {
      if (cur == null) return null;
      cur = cur[p];
    }
    return (cur === undefined) ? null : cur;
  }

  globalThis.log = {
    parseJSON,
    parseLogfmt,
    namedCapture,
    extract,
    field,
  };
})();
`
