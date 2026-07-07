# IRAN DISCORD — Stats Bot

## English

This bot is designed to be completely safe to add to your server.

**It runs with ZERO permissions.** When you invite it, it does not ask for any powers on your server — it cannot send messages, delete anything, kick or ban members, manage channels, or change any settings. It simply exists in the server and reads public information.

**What it does:**
- Counts how many members your server has
- Counts how many members are currently online
- Sends only these numbers to our website so your server's stats can be displayed

**What it does NOT do:**
- It does not read your messages
- It does not store any personal information about your members
- It does not collect emails, IDs, or private data
- It does not have access to private channels
- It cannot make any changes to your server

**Why you can trust it:**
The full source code of this bot is public in this repository. Anyone can read every line and confirm exactly what it does. There is nothing hidden. Because the bot has no permissions, even in the worst case it cannot harm your server in any way.

**Run it yourself (optional):**
If you want to be completely certain, you can run the bot on your own machine instead of trusting a hosted version. This way you know the running bot is exactly the code you just read.

1. Install Go 1.22 or newer.
2. Copy `.env.example` to `.env` and fill in your own values.
3. Build and run:

```
go build -o bin/iran-discord-stats ./cmd/bot
./bin/iran-discord-stats
```

On Windows:

```
go build -o bin\iran-discord-stats.exe .\cmd\bot
.\bin\iran-discord-stats.exe
```

When the bot starts, it prints an invite link with **zero permissions** that you can use to add your own copy to a server.

---

## فارسی

این ربات به‌گونه‌ای طراحی شده که افزودن آن به سرور شما کاملاً بی‌خطر باشد.

**این ربات با صفر دسترسی (Zero Permissions) کار می‌کند.** وقتی آن را دعوت می‌کنید، هیچ دسترسی‌ای روی سرور شما نمی‌خواهد — نمی‌تواند پیام بفرستد، چیزی را حذف کند، کسی را اخراج یا بن کند، کانال‌ها را مدیریت کند یا هیچ تنظیمی را تغییر دهد. فقط در سرور حضور دارد و اطلاعات عمومی را می‌خواند.

**این ربات چه کاری انجام می‌دهد:**
- تعداد اعضای سرور شما را می‌شمارد
- تعداد اعضای آنلاین را می‌شمارد
- فقط همین اعداد را برای نمایش آمار سرور به وب‌سایت ما ارسال می‌کند

**این ربات چه کاری انجام نمی‌دهد:**
- پیام‌های شما را نمی‌خواند
- هیچ اطلاعات شخصی از اعضا ذخیره نمی‌کند
- ایمیل، آیدی یا اطلاعات خصوصی جمع‌آوری نمی‌کند
- به کانال‌های خصوصی دسترسی ندارد
- نمی‌تواند هیچ تغییری در سرور شما ایجاد کند

**چرا می‌توانید به آن اعتماد کنید:**
کل سورس‌کد این ربات در همین ریپازیتوری به‌صورت عمومی در دسترس است. هر کسی می‌تواند تک‌تک خطوط آن را بخواند و دقیقاً ببیند که ربات چه کاری انجام می‌دهد. هیچ چیز پنهانی وجود ندارد. چون ربات هیچ دسترسی‌ای ندارد، حتی در بدترین حالت هم نمی‌تواند به سرور شما آسیبی برساند.

**اجرای شخصی (اختیاری):**
اگر می‌خواهید کاملاً مطمئن شوید، می‌توانید ربات را به‌جای اعتماد به نسخهٔ میزبانی‌شده، روی سیستم خودتان اجرا کنید. به این ترتیب مطمئن می‌شوید ربات در حال اجرا دقیقاً همان کدی است که خواندید.

۱. نسخهٔ Go 1.22 یا جدیدتر را نصب کنید.
۲. فایل `.env.example` را به `.env` کپی کنید و مقادیر خودتان را وارد کنید.
۳. بیلد و اجرا کنید:

```
go build -o bin/iran-discord-stats ./cmd/bot
./bin/iran-discord-stats
```

در ویندوز:

```
go build -o bin\iran-discord-stats.exe .\cmd\bot
.\bin\iran-discord-stats.exe
```

هنگام اجرا، ربات یک لینک دعوت با **صفر دسترسی** نمایش می‌دهد که می‌توانید با آن نسخهٔ خودتان را به یک سرور اضافه کنید.
