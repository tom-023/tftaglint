# tftaglint

tftaglint は Terraform リソースのタグを検証するためのツールです。設定ファイルで定義されたルールに基づいて、タグの組み合わせや値を検証し、違反があればファイル名と行番号と共に報告します。

## 特徴

- 🏷️ Terraformリソースのタグを包括的に検証
- 📝 YAMLベースの柔軟なルール設定
- 📍 違反箇所の正確な位置情報（ファイル名・行番号）を出力
- 🎯 リソースタイプ別のルール設定が可能
- ⚡ 高速な解析処理

## インストール

```bash
go install github.com/tom-023/tftaglint/cmd/tftaglint@latest
```

または、ソースからビルド：

```bash
git clone https://github.com/tom-023/tftaglint.git
cd tftaglint
go build -o tftaglint cmd/tftaglint/main.go
```

## 使い方

### 基本的な使い方

```bash
# カレントディレクトリのTerraformファイルを検証
tftaglint validate

# 特定のディレクトリを検証
tftaglint validate ./terraform/

# カスタム設定ファイルを使用
tftaglint validate -c custom-rules.yaml

# -f オプションでもカスタム設定ファイルを指定可能（-c のエイリアス）
tftaglint validate -f my-tag-rules.yaml

# サマリーも表示
tftaglint validate -s
```

### Terraform Plan を使った検証（推奨）

`locals`や変数を使ってタグを管理している場合は、terraform planの出力を使用することで、実際に適用される値で検証できます。

```bash
# terraform planをJSON形式で出力
terraform plan -out=tfplan
terraform show -json tfplan > tfplan.json

# planファイルを使って検証
tftaglint validate --plan tfplan.json

# または短縮形
tftaglint validate -p tfplan.json -s
```

この方法の利点：
- 変数展開後の実際の値で検証
- モジュール内のリソースも検証対象
- `locals`で定義されたタグも正しく認識

## 設定ファイル

tftaglintは `tag-rules.yaml` という設定ファイルでルールを定義します。

### 設定例

```yaml
rules:
  # 必須タグの定義
  - name: "environment-required"
    description: "すべてのリソースにはEnvironmentタグが必要です"
    required_tags:
      - Environment

  # 条件付きルール
  - name: "production-tags"
    description: "本番環境のリソースには追加のタグが必要です"
    condition:
      tag: Environment
      value: production
    required_tags:
      - Owner
      - CostCenter
      - BackupRequired

  # 禁止タグの定義
  - name: "no-test-in-production"
    description: "本番環境でTestタグは使用できません"
    condition:
      tag: Environment
      value: production
    forbidden_tags:
      - Test
      - Temporary

  # タグ値の検証
  - name: "valid-environment-values"
    description: "Environmentタグの値は定義された値のみ許可"
    tag_constraints:
      - tag: Environment
        allowed_values:
          - development
          - staging
          - production

# グローバル設定
global:
  always_required_tags:
    - Project
    - ManagedBy
  ignore_resource_types:
    - data.aws_ami
```

## ルールの種類

### 1. 必須タグ (`required_tags`)
指定されたタグが存在することを要求します。

### 2. 禁止タグ (`forbidden_tags`)
指定されたタグが存在しないことを要求します。

### 3. 条件付きルール (`condition`)
特定のタグと値の組み合わせが存在する場合にのみ、ルールを適用します。

### 4. タグ制約 (`tag_constraints`)
タグの値が許可されたリストに含まれることを検証します。

### 5. タグパターン (`tag_patterns`)
タグ名が正規表現パターンに一致することを検証します。

### 6. リソースタイプ別ルール (`resource_types`)
特定のリソースタイプにのみルールを適用します。

## 出力例

```
❌ Found 4 tag violation(s):

📄 test_data/example.tf
  Line 15: aws_instance.db
    Rule: no-test-in-production
    Message: Forbidden tag found: Test
    Description: 本番環境でTestタグは使用できません

  Line 15: aws_instance.db
    Rule: global-required-tags
    Message: Missing required tag: ManagedBy
    Description: Global required tags

  Line 26: aws_s3_bucket.logs
    Rule: global-required-tags
    Message: Missing required tag: ManagedBy
    Description: Global required tags

  Line 37: aws_instance.test
    Rule: valid-environment-values
    Message: Invalid value for tag Environment: 'invalid-env'. Allowed values: development, staging, production
    Description: Environmentタグの値は定義された値のみ許可
```

## ライセンス

MIT License