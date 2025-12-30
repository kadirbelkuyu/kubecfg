# Git Workflow Cheat Sheet

Bu, KubeConfigCli projesi için hızlı başvuru kartıdır. Detaylı açıklama için [GIT_WORKFLOW.md](./GIT_WORKFLOW.md) belgesine bakın.

---

## 🚀 Yeni Feature Eklemek

```bash
# 1. Feature branch oluştur
git checkout main
git pull origin main
git checkout -b feature/feature-name

# 2. Geliştir ve commit et
git add .
git commit -m "feat: feature açıklaması"

# 3. Push et
git push -u origin feature/feature-name

# 4. Main'e merge et
git checkout main
git merge feature/feature-name --no-ff
git push origin main
```

---

## 🐛 Bug Fix

```bash
# 1. Bugfix branch oluştur
git checkout -b bugfix/bug-name

# 2. Fix ve commit
git add .
git commit -m "fix: bug açıklaması"

# 3. Main'e merge et
git checkout main
git merge bugfix/bug-name --no-ff
git push origin main
```

---

## 🏷️ Release Çıkarmak

```bash
# 1. Version belirle: v0.0.5 (patch), v0.1.0 (minor), v1.0.0 (major)
# 2. Tag oluştur
git checkout main
git pull origin main
git tag -a v0.0.5 -m "Release v0.0.5 - Özet

Features:
- ...

Bug Fixes:
- ...
"

# 3. Push et
git push origin v0.0.5
```

---

## 📝 Commit Mesaj Formatı

```
<type>: <kısa açıklama>

<detaylı açıklama (opsiyonel)>

<footer (opsiyonel)>
```

### Commit Types

- `feat`: Yeni özellik
- `fix`: Bug fix
- `docs`: Dokümantasyon
- `style`: Formatting
- `refactor`: Refactoring
- `test`: Test
- `chore`: Build/tooling
- `perf`: Performance
- `ci`: CI/CD

### Örnekler

```bash
git commit -m "feat: add user authentication"
git commit -m "fix: resolve null pointer in login"
git commit -m "docs: update installation guide"
git commit -m "refactor: simplify error handling"
```

---

## 📊 Version Numarası (Semantic Versioning)

```
vMAJOR.MINOR.PATCH

v1.2.3
 │ │ └─ Patch: Bug fixes
 │ └─── Minor: Yeni özellikler (backward compatible)
 └───── Major: Breaking changes
```

### Ne Zaman Artırılır?

| Değişiklik | Version | Örnek |
|------------|---------|-------|
| Bug fix, küçük iyileştirme | PATCH | v1.0.0 → v1.0.1 |
| Yeni özellik (uyumlu) | MINOR | v1.0.0 → v1.1.0 |
| Breaking change | MAJOR | v1.0.0 → v2.0.0 |

---

## 🔧 Sık Kullanılan Komutlar

```bash
# Durum kontrolü
git status
git log --oneline -5
git branch -a

# Branch değiştirme
git checkout branch-name
git checkout -b new-branch-name

# Değişiklikleri geri alma
git reset --soft HEAD~1      # Son commit'i geri al (değişiklikler kalır)
git reset --hard HEAD~1       # Son commit'i tamamen geri al
git restore file-name         # Dosya değişikliklerini geri al

# Remote işlemleri
git pull origin main
git push origin branch-name
git push origin --tags

# Branch temizleme
git branch -d branch-name                    # Lokal branch sil
git push origin --delete branch-name          # Remote branch sil

# Tag işlemleri
git tag -l                                    # Tag listesi
git tag -d tag-name                           # Lokal tag sil
git push origin :refs/tags/tag-name           # Remote tag sil
```

---

## ⚠️ Unutmayın

- ✅ Her zaman `--no-ff` ile merge edin
- ✅ Conventional commits kullanın
- ✅ Annotated tag oluşturun (`-a` flag)
- ✅ Main'e direkt commit atmayın
- ✅ Meaningful branch isimleri kullanın

---

## 🆘 Acil Durum

### Yanlış branch'de commit attım

```bash
git reset --soft HEAD~1
git checkout correct-branch
git commit -m "message"
```

### Main'e yanlışlıkla direkt commit attım

```bash
git reset --soft HEAD~1
git checkout -b feature/my-feature
git commit -m "message"
git push -u origin feature/my-feature
```

### Merge conflict

```bash
git status                    # Çakışan dosyaları gör
# Dosyaları manuel düzenle
git add resolved-file
git commit
```

---

**Detaylı bilgi**: [GIT_WORKFLOW.md](./GIT_WORKFLOW.md)
