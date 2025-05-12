const path = require('path');

module.exports = {
  entry: './public/index.js',
  output: {
    filename: 'bundle.js',
    path: path.resolve(__dirname, 'build'),
  },
  mode: 'production',
};