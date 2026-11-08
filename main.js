// 不能使用 import/export (这里作为无打包器的普通 script 运行)。
// 需要拆分文件时，在 index.html 中添加 <script> 标签来加载。
var count = 0;
var button = document.getElementById('counter');
button.addEventListener('click', function () {
  count += 1;
  button.textContent = '点击: ' + count;
});
