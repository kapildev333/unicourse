// Same logic as boot() in src/lib/theme.ts: apply the stored theme before first paint. A file, not inline: the CSP forbids inline scripts.
try{var t=localStorage.getItem('unicourse:theme');if(t==='light'||t==='dark')document.documentElement.setAttribute('data-theme',t)}catch(e){}
