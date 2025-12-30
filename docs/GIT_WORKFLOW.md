# Git Workflow & Release Process

Bu dokümantasyon, KubeConfigCli projesi için standart Git workflow ve release süreçlerini açıklar. Her yeni feature veya bug fix için bu adımları izleyin.

---

## 📋 İçindekiler

- [Genel Prensipler](#genel-prensipler)
- [Branch Stratejisi](#branch-stratejisi)
- [Feature Geliştirme](#feature-geliştirme)
- [Commit Standartları](#commit-standartları)
- [Release Süreci](#release-süreci)
- [Semantic Versioning](#semantic-versioning)
- [Hızlı Başvuru](#hızlı-başvuru)

---

## 🎯 Genel Prensipler

### Best Practices

1. **Asla doğrudan `main` branch'e commit atmayın**
2. **Feature branch'lerini kullanın**
3. **Conventional Commits standardını takip edin**
4. **No Fast-Forward merge kullanın** (`--no-ff`)
5. **Annotated tag'ler oluşturun**
6. **Semantic Versioning uygulayın**

### Branch Yapısı

```
main (protected)
  ├── feature/feature-name
  ├── bugfix/bug-name
  └── hotfix/critical-fix
```

---

## 🌿 Branch Stratejisi

### Branch İsimlendirme Kuralları

| Tip | Prefix | Örnek |
|-----|--------|-------|
| Yeni özellik | `feature/` | `feature/user-authentication` |
| Hata düzeltme | `bugfix/` | `bugfix/login-error` |
| Acil düzeltme | `hotfix/` | `hotfix/security-patch` |
| Dokümantasyon | `docs/` | `docs/api-documentation` |
| Refactoring | `refactor/` | `refactor/code-cleanup` |

### İsimlendirme İpuçları

- **Kebab-case kullanın**: `feature/my-new-feature`
- **Açıklayıcı olun**: `feature/add-oauth-support`
- **Kısa ve öz tutun**: `bugfix/fix-namespace-error`
- **Issue numarası ekleyin**: `feature/123-user-profiles`

---

## 🚀 Feature Geliştirme

### Adım 1: Mevcut Durumu Kontrol Edin

```bash
# Main branch'e geçin
git checkout main

# Güncel hale getirin
git pull origin main

# Mevcut branch'leri görüntüleyin
git branch -a
```

### Adım 2: Feature Branch Oluşturun

```bash
# Yeni branch oluşturun ve geçin
git checkout -b feature/your-feature-name

# Örnek:
git checkout -b feature/ui-enhancement
```

### Adım 3: Geliştirme Yapın

```bash
# Kodunuzu yazın, testlerinizi yapın
# Değişiklikleri düzenli olarak commit edin
```

### Adım 4: Değişiklikleri Commit Edin

```bash
# Değişiklikleri görüntüleyin
git status

# Tüm değişiklikleri ekleyin
git add .

# Veya seçici olarak ekleyin
git add path/to/file

# Conventional commit ile commit edin (aşağıda detay)
git commit -m "feat: add new feature description"
```

### Adım 5: Remote'a Push Edin

```bash
# İlk push (upstream set eder)
git push -u origin feature/your-feature-name

# Sonraki push'lar
git push
```

### Adım 6: Main'e Merge Edin

```bash
# Main branch'e geçin
git checkout main

# No fast-forward ile merge edin
git merge feature/your-feature-name --no-ff -m "Merge feature/your-feature-name into main

Brief description of the feature"

# Remote'a push edin
git push origin main
```

### Adım 7: Feature Branch'i Temizleyin (Opsiyonel)

```bash
# Lokal branch'i silin
git branch -d feature/your-feature-name

# Remote branch'i silin
git push origin --delete feature/your-feature-name
```

---

## 📝 Commit Standartları

### Conventional Commits Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Commit Tipleri

| Tip | Açıklama | Örnek |
|-----|----------|-------|
| `feat` | Yeni özellik | `feat: add user login` |
| `fix` | Hata düzeltme | `fix: resolve null pointer error` |
| `docs` | Dokümantasyon | `docs: update README` |
| `style` | Code style (formatting) | `style: fix indentation` |
| `refactor` | Code refactoring | `refactor: simplify auth logic` |
| `test` | Test ekleme/düzeltme | `test: add unit tests for auth` |
| `chore` | Build, tooling | `chore: update dependencies` |
| `perf` | Performance iyileştirme | `perf: optimize database queries` |
| `ci` | CI/CD değişiklikleri | `ci: add github actions` |

### Commit Message Örnekleri

#### Basit Commit

```bash
git commit -m "feat: add namespace selection feature"
```

#### Detaylı Commit

```bash
git commit -m "feat: enhance terminal UI with lipgloss

- Add lipgloss library for beautiful terminal styling
- Create comprehensive UI styles package with colors and icons
- Enhance all commands with styled output:
  - list: Beautiful table with colored headers and separators
  - use: Enhanced interactive selector with icons
  - ns: Improved namespace selection with styled prompts
  - current: Detailed context information card

BREAKING CHANGE: None
Closes: #123"
```

#### Breaking Change Commit

```bash
git commit -m "feat!: change API authentication method

BREAKING CHANGE: API now requires OAuth 2.0 instead of API keys.
Migrate by following the migration guide in docs/migration.md

Closes: #456"
```

### Scope Kullanımı (Opsiyonel)

```bash
feat(ui): add dark mode support
fix(auth): resolve token expiration issue
docs(api): update endpoint documentation
```

---

## 🏷️ Release Süreci

### Adım 1: Version Numarasını Belirleyin

[Semantic Versioning](#semantic-versioning) bölümüne bakın.

Örnek: `v0.0.5`, `v0.1.0`, `v1.0.0`

### Adım 2: Main Branch'in Güncel Olduğundan Emin Olun

```bash
git checkout main
git pull origin main
```

### Adım 3: Annotated Tag Oluşturun

```bash
git tag -a v0.0.5 -m "Release v0.0.5 - Feature Name

Features:
- Feature 1 description
- Feature 2 description
- Feature 3 description

Improvements:
- Improvement 1
- Improvement 2

Bug Fixes:
- Bug fix 1
- Bug fix 2

Dependencies:
- Added dependency-name v1.2.3
- Updated dependency-name to v2.0.0

Breaking Changes:
- None (veya breaking change açıklaması)
"
```

### Adım 4: Tag'i Remote'a Push Edin

```bash
# Tek tag push
git push origin v0.0.5

# Veya tüm tag'leri push
git push origin --tags
```

### Adım 5: GitHub Release'i Kontrol Edin

- GitHub'da `Releases` bölümüne gidin
- GoReleaser otomatik olarak release oluşturur
- Build artifact'ları ve release notes'u kontrol edin

---

## 📊 Semantic Versioning

Format: `MAJOR.MINOR.PATCH` (örn: `v1.2.3`)

### Version Artırma Kuralları

| Version | Ne Zaman Artırılır | Örnek |
|---------|-------------------|-------|
| **MAJOR** | Breaking changes | `v1.0.0` → `v2.0.0` |
| **MINOR** | Yeni özellikler (backward compatible) | `v1.0.0` → `v1.1.0` |
| **PATCH** | Bug fixes, küçük iyileştirmeler | `v1.0.0` → `v1.0.1` |

### Version Örnekleri

```
v0.0.1  → İlk development release
v0.0.5  → Birkaç patch sonrası
v0.1.0  → İlk minor özellik eklemesi
v0.2.0  → Başka bir minor özellik
v1.0.0  → İlk production-ready release
v1.0.1  → Bug fix
v1.1.0  → Yeni özellik (backward compatible)
v2.0.0  → Breaking change
```

### Pre-release Versioning (Opsiyonel)

```
v1.0.0-alpha.1    → Alpha release
v1.0.0-beta.1     → Beta release
v1.0.0-rc.1       → Release candidate
v1.0.0            → Final release
```

---

## ⚡ Hızlı Başvuru

### Yeni Feature

```bash
# 1. Main'den başla
git checkout main
git pull origin main

# 2. Feature branch oluştur
git checkout -b feature/my-feature

# 3. Geliştirme yap ve commit et
git add .
git commit -m "feat: add my feature description"

# 4. Push et
git push -u origin feature/my-feature

# 5. Main'e merge et
git checkout main
git merge feature/my-feature --no-ff -m "Merge feature/my-feature into main"
git push origin main
```

### Bug Fix

```bash
# 1. Bugfix branch oluştur
git checkout -b bugfix/fix-issue-name

# 2. Fix yap ve commit et
git add .
git commit -m "fix: resolve issue description"

# 3. Main'e merge et
git checkout main
git merge bugfix/fix-issue-name --no-ff
git push origin main
```

### Release

```bash
# 1. Main'in güncel olduğundan emin ol
git checkout main
git pull origin main

# 2. Tag oluştur
git tag -a v0.1.0 -m "Release v0.1.0 - Description

Features:
- Feature list

Bug Fixes:
- Fix list
"

# 3. Tag'i push et
git push origin v0.1.0
```

---

## 🔧 Troubleshooting

### Merge Conflict Çözümü

```bash
# Conflict durumunda
git status  # Conflicted files'ı görün

# Her conflicted file'ı düzenleyin
# Conflict markers'ları temizleyin: <<<<<<<, =======, >>>>>>>

# Çözümü stage edin
git add path/to/resolved/file

# Merge'i tamamlayın
git commit
```

### Yanlış Branch'de Commit Attınız

```bash
# Son commit'i geri al (değişiklikler working directory'de kalır)
git reset --soft HEAD~1

# Doğru branch'e geçin
git checkout correct-branch

# Tekrar commit edin
git commit -m "your message"
```

### Tag Silme

```bash
# Lokal tag'i sil
git tag -d v0.0.5

# Remote tag'i sil
git push origin :refs/tags/v0.0.5
```

---

## 📚 Kaynaklar

- [Conventional Commits](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [Git Flow](https://nvie.com/posts/a-successful-git-branching-model/)
- [GitHub Flow](https://guides.github.com/introduction/flow/)

---

## ✅ Checklist

Her release öncesi kontrol edin:

- [ ] Tüm testler geçiyor mu?
- [ ] Kodunuz build ediliyor mu?
- [ ] README güncel mi?
- [ ] CHANGELOG güncellendi mi? (varsa)
- [ ] Version numarası doğru mu?
- [ ] Commit messages conventional standarda uygun mu?
- [ ] Tag message detaylı mı?
- [ ] Breaking changes dokümante edildi mi?

---

**Son Güncelleme**: 2025-12-30
**Versiyon**: 1.0.0
