'use strict'

var preload = function () {
  const ele = document.getElementById('loading');
  const params = new URLSearchParams(window.location.search);
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
  document.cookie = `vscode=${params.get('cookie')}; path=/`;
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
    if (params.get('debug')) {
    } else {
      setTimeout(function () {
        window.location.href = '/';
      }, 500)
    }

  });
}


document.addEventListener("DOMContentLoaded", preload);