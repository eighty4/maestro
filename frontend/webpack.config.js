const path = require('path')
const HtmlWebpackPlugin = require('html-webpack-plugin')
const CopyWebpackPlugin = require('copy-webpack-plugin')

module.exports = {
    devServer: {
        port: 2999,
    },
    entry: './src/Maestro.jsx',
    output: {
        path: path.join(process.cwd(), 'dist'),
        filename: 'maestro.js',
    },
    module: {
        rules: [
            {
                test: /\.(js|jsx)$/,
                exclude: /node_modules/,
                loader: 'babel-loader',
            }
        ],
    },
    plugins: [
        new HtmlWebpackPlugin({
            title: 'Maestro',
            template: 'src/index.html',
        }),
        new CopyWebpackPlugin({
            patterns: [
                {from: 'static'},
            ]
        }),
    ],
}
