'use strict'

var preload = function () {
  const ele = document.getElementById('loading');
  const btnContinue = document.getElementById('btn-continue');
  const params = new URLSearchParams(window.location.search);
  const vscodeCookie = params.get('vscode');

  const largeFiles = [
    // file to download, estimated size in bytes
    ['/static/out/vs/workbench/workbench.web.main.js', 11 * 1024 * 1024],
    ['/static/out/vs/workbench/api/worker/extensionHostWorker.js', 1 * 1024 * 1024],
    ['/static/out/vs/workbench/workbench.web.main.nls.js', 500 * 1024],
    ['/static/node_modules/vscode-oniguruma/release/onig.wasm', 500 * 1024],
    ['/static/out/vs/base/worker/workerMain.js', 300 * 1024],
  ];

  let totalSize = 0;
  let loadedMap = new Map();
  document.cookie = `vscode=${vscodeCookie}; path=/`;
  const updateLoaded = function (file, loaded) {
    loadedMap.set(file, loaded);
    let loadedSize = 0;
    loadedMap.forEach((loaded, _) => {
      loadedSize += loaded;
    });
    ele.textContent = `Loading: ${Math.floor(loadedSize / totalSize * 100)}%`;
  }

  largeFiles.forEach(([file, size]) => {
    totalSize += size;
  });

  // copy from: https://stackoverflow.com/a/62979491
  const iPad = !!(navigator.userAgent.match(/(iPad)/)
    || (navigator.platform === "MacIntel" && typeof navigator.standalone !== "undefined"))
  // copy from: https://stackoverflow.com/a/52695341
  const isInStandaloneMode = () =>
    (window.matchMedia('(display-mode: standalone)').matches) || (window.navigator.standalone) || document.referrer.includes('android-app://');

  const rewritePage = function () {
    // you can't use location.href if the page is added to the homescreen in iPad safari.
    // iPad will display an ugly top bar with an url, so we use document.write to replace the page with the new one
    // It can also used in the standalone mode in PC browser.
    const xhr = new XMLHttpRequest();
    xhr.onreadystatechange = () => {
      let data = xhr.responseText;
      document.write(data);
    }
    xhr.open('GET', '/', true);
    xhr.send();
  };

  Promise.all(
    largeFiles.map(([file, _]) =>
      new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.onprogress = (event) => updateLoaded(file, event.loaded);
        xhr.onload = () => resolve()
        xhr.onerror = () => reject(`Download ${file} failed.`);
        xhr.onabort = () => reject(`Download ${file} cancelled.`);
        xhr.withCredentials = true;
        xhr.open('GET', file);
        xhr.send();
      })
    )
  ).then(() => {
    ele.textContent = 'Load success';

    // debug mode, do nothing
    if (params.get('debug')) {
      return;
    }

    // browser PWA, only rewrite the page
    if (isInStandaloneMode()) {
      rewritePage();
      return;
    }

    btnContinue.style.display = "block";
    btnContinue.onclick = () => { window.location.href = `/?vscode=${vscodeCookie}` };
  });
}


document.addEventListener("DOMContentLoaded", preload);