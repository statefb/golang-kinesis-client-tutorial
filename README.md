# Golang kinesis consumer をさわってみる

## Go の consumer

- ライブラリは二つ
  - [Kinesis Consumer](https://github.com/harlow/kinesis-consumer)
    - multiple consumer に未対応。シャードごとに別のコンシューマーに消費させることが基本的にできない
      - https://github.com/harlow/kinesis-consumer/issues/114
      - https://github.com/harlow/kinesis-consumer/issues/42
      - 単一の consumer ですべてのシャードを watch している
        - 実装: https://github.com/harlow/kinesis-consumer/blob/master/allgroup.go#L13
    - シャード ID を指定する独自実装すれば multiple consumer が可能
      - auto scale 時にテーブルを際紐付けする機能はない
      - provisioned でシャード数が固定の場合なら良いのか・・？
  - [Kinsumer](https://github.com/twitchscience/kinsumer/tree/master)
    - こちらは multiple consumer に対応
    - 本リポジトリで利用

## Getting started

- `cdk deploy`
- kinesis generator 使って送信
- cloudwatch logs tail で出力を眺める
