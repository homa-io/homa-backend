/**
 * Homa Chat Widget SDK
 * Version: 1.1.10 (2026-01-04)
 *
 * A customizable chat widget that integrates with the Homa support backend.
 * Inspired by Chatwoot, Intercom, and Crisp best practices.
 *
 * Usage:
 *   <script>
 *     (function(w,d,s,o,f,js,fjs){
 *       w['HomaChat']=o;w[o]=w[o]||function(){(w[o].q=w[o].q||[]).push(arguments)};
 *       js=d.createElement(s);fjs=d.getElementsByTagName(s)[0];
 *       js.id=o;js.src=f;js.async=1;fjs.parentNode.insertBefore(js,fjs);
 *     })(window,document,'script','homaChat','https://your-domain.com/widget/homa-chat.js');
 *
 *     homaChat('init', {
 *       websiteToken: 'YOUR_WEBSITE_TOKEN',
 *       baseUrl: 'https://api.example.com',
 *       locale: 'auto', // Auto-detect or specify: en, es, fr, de, etc.
 *       // Customization options:
 *       launcherSize: 60,
 *       launcherIcon: null, // Custom SVG string
 *       brandColor: '#3B82F6',
 *       headerColor: '#3B82F6',
 *       windowWidth: 380,
 *       windowHeight: 600,
 *       borderRadius: 16,
 *       // Pre-fill user info:
 *       userName: 'John Doe',
 *       userEmail: 'john@example.com',
 *       userAttributes: { plan: 'premium' },
 *       conversationAttributes: { source: 'pricing_page' }
 *     });
 *   </script>
 */

(function(window, document) {
  'use strict';

  // ==========================================
  // Internationalization (i18n)
  // ==========================================
  const i18n = {
    en: {
      greetingTitle: 'Welcome',
      greetingMessage: 'Hi! How can we help you today?',
      launcherText: 'Chat with us',
      replyTime: 'We typically reply within a few minutes',
      nameLabel: 'Name',
      nameRequired: '*',
      namePlaceholder: 'Enter your full name',
      emailLabel: 'Email',
      emailOptional: '(optional)',
      emailPlaceholder: 'For follow-up communication',
      phoneLabel: 'Phone',
      phonePlaceholder: 'Your phone number',
      companyLabel: 'Company',
      companyPlaceholder: 'Your company name',
      departmentLabel: 'Department',
      departmentPlaceholder: 'Select a department...',
      messageLabel: 'How can we help?',
      messagePlaceholder: 'Describe your question or issue...',
      startChat: 'Start Chat',
      startingChat: 'Starting chat...',
      sendMessage: 'Send message',
      typeMessage: 'Type your message...',
      poweredBy: 'Powered by',
      you: 'You',
      endChat: 'End chat',
      endChatConfirm: 'Are you sure you want to end this chat?',
      offlineTitle: 'We are offline',
      offlineDefault: 'We are currently offline. Please leave a message and we will get back to you.',
      emailTranscript: 'Email transcript'
    },
    es: {
      greetingTitle: 'Bienvenido',
      greetingMessage: '¡Hola! ¿Cómo podemos ayudarte hoy?',
      launcherText: 'Chatea con nosotros',
      replyTime: 'Normalmente respondemos en pocos minutos',
      nameLabel: 'Nombre',
      nameRequired: '*',
      namePlaceholder: 'Ingresa tu nombre completo',
      emailLabel: 'Correo electrónico',
      emailOptional: '(opcional)',
      emailPlaceholder: 'Para comunicación de seguimiento',
      departmentLabel: 'Departamento',
      departmentPlaceholder: 'Selecciona un departamento...',
      messageLabel: '¿Cómo podemos ayudarte?',
      messagePlaceholder: 'Describe tu pregunta o problema...',
      startChat: 'Iniciar Chat',
      startingChat: 'Iniciando chat...',
      sendMessage: 'Enviar mensaje',
      typeMessage: 'Escribe tu mensaje...',
      poweredBy: 'Desarrollado por',
      you: 'Tú',
      endChat: 'Terminar chat',
      endChatConfirm: '¿Estás seguro de que quieres terminar este chat?'
    },
    fr: {
      greetingTitle: 'Bienvenue',
      greetingMessage: 'Bonjour! Comment pouvons-nous vous aider?',
      launcherText: 'Discutez avec nous',
      replyTime: 'Nous répondons généralement en quelques minutes',
      nameLabel: 'Nom',
      nameRequired: '*',
      namePlaceholder: 'Entrez votre nom complet',
      emailLabel: 'E-mail',
      emailOptional: '(facultatif)',
      emailPlaceholder: 'Pour la communication de suivi',
      departmentLabel: 'Département',
      departmentPlaceholder: 'Sélectionnez un département...',
      messageLabel: 'Comment pouvons-nous vous aider?',
      messagePlaceholder: 'Décrivez votre question ou problème...',
      startChat: 'Démarrer le Chat',
      startingChat: 'Démarrage du chat...',
      sendMessage: 'Envoyer le message',
      typeMessage: 'Tapez votre message...',
      poweredBy: 'Propulsé par',
      you: 'Vous',
      endChat: 'Terminer le chat',
      endChatConfirm: 'Êtes-vous sûr de vouloir terminer ce chat?'
    },
    de: {
      greetingTitle: 'Willkommen',
      greetingMessage: 'Hallo! Wie können wir Ihnen heute helfen?',
      launcherText: 'Mit uns chatten',
      replyTime: 'Wir antworten normalerweise innerhalb weniger Minuten',
      nameLabel: 'Name',
      nameRequired: '*',
      namePlaceholder: 'Geben Sie Ihren vollständigen Namen ein',
      emailLabel: 'E-Mail',
      emailOptional: '(optional)',
      emailPlaceholder: 'Für die Nachverfolgung',
      departmentLabel: 'Abteilung',
      departmentPlaceholder: 'Wählen Sie eine Abteilung...',
      messageLabel: 'Wie können wir helfen?',
      messagePlaceholder: 'Beschreiben Sie Ihre Frage oder Ihr Problem...',
      startChat: 'Chat Starten',
      startingChat: 'Chat wird gestartet...',
      sendMessage: 'Nachricht senden',
      typeMessage: 'Ihre Nachricht eingeben...',
      poweredBy: 'Unterstützt von',
      you: 'Sie',
      endChat: 'Chat beenden',
      endChatConfirm: 'Sind Sie sicher, dass Sie diesen Chat beenden möchten?'
    },
    it: {
      greetingTitle: 'Benvenuto',
      greetingMessage: 'Ciao! Come possiamo aiutarti oggi?',
      launcherText: 'Chatta con noi',
      replyTime: 'Di solito rispondiamo in pochi minuti',
      nameLabel: 'Nome',
      nameRequired: '*',
      namePlaceholder: 'Inserisci il tuo nome completo',
      emailLabel: 'E-mail',
      emailOptional: '(facoltativo)',
      emailPlaceholder: 'Per la comunicazione di follow-up',
      departmentLabel: 'Dipartimento',
      departmentPlaceholder: 'Seleziona un dipartimento...',
      messageLabel: 'Come possiamo aiutarti?',
      messagePlaceholder: 'Descrivi la tua domanda o problema...',
      startChat: 'Inizia Chat',
      startingChat: 'Avvio chat...',
      sendMessage: 'Invia messaggio',
      typeMessage: 'Scrivi il tuo messaggio...',
      poweredBy: 'Offerto da',
      you: 'Tu',
      endChat: 'Termina chat',
      endChatConfirm: 'Sei sicuro di voler terminare questa chat?'
    },
    pt: {
      greetingTitle: 'Bem-vindo',
      greetingMessage: 'Olá! Como podemos ajudá-lo hoje?',
      launcherText: 'Fale conosco',
      replyTime: 'Normalmente respondemos em poucos minutos',
      nameLabel: 'Nome',
      nameRequired: '*',
      namePlaceholder: 'Digite seu nome completo',
      emailLabel: 'E-mail',
      emailOptional: '(opcional)',
      emailPlaceholder: 'Para comunicação de acompanhamento',
      departmentLabel: 'Departamento',
      departmentPlaceholder: 'Selecione um departamento...',
      messageLabel: 'Como podemos ajudar?',
      messagePlaceholder: 'Descreva sua pergunta ou problema...',
      startChat: 'Iniciar Chat',
      startingChat: 'Iniciando chat...',
      sendMessage: 'Enviar mensagem',
      typeMessage: 'Digite sua mensagem...',
      poweredBy: 'Desenvolvido por',
      you: 'Você',
      endChat: 'Encerrar chat',
      endChatConfirm: 'Tem certeza de que deseja encerrar este chat?'
    },
    ru: {
      greetingTitle: 'Добро пожаловать',
      greetingMessage: 'Привет! Как мы можем вам помочь?',
      launcherText: 'Написать нам',
      replyTime: 'Обычно отвечаем в течение нескольких минут',
      nameLabel: 'Имя',
      nameRequired: '*',
      namePlaceholder: 'Введите ваше полное имя',
      emailLabel: 'Эл. почта',
      emailOptional: '(необязательно)',
      emailPlaceholder: 'Для обратной связи',
      departmentLabel: 'Отдел',
      departmentPlaceholder: 'Выберите отдел...',
      messageLabel: 'Чем мы можем помочь?',
      messagePlaceholder: 'Опишите ваш вопрос или проблему...',
      startChat: 'Начать чат',
      startingChat: 'Запуск чата...',
      sendMessage: 'Отправить сообщение',
      typeMessage: 'Введите сообщение...',
      poweredBy: 'Работает на',
      you: 'Вы',
      endChat: 'Завершить чат',
      endChatConfirm: 'Вы уверены, что хотите завершить этот чат?'
    },
    zh: {
      greetingTitle: '欢迎',
      greetingMessage: '您好！我们今天能为您提供什么帮助？',
      launcherText: '在线咨询',
      replyTime: '我们通常会在几分钟内回复',
      nameLabel: '姓名',
      nameRequired: '*',
      namePlaceholder: '请输入您的全名',
      emailLabel: '电子邮箱',
      emailOptional: '(可选)',
      emailPlaceholder: '用于后续沟通',
      departmentLabel: '部门',
      departmentPlaceholder: '选择部门...',
      messageLabel: '我们能帮您什么？',
      messagePlaceholder: '描述您的问题...',
      startChat: '开始聊天',
      startingChat: '正在启动...',
      sendMessage: '发送消息',
      typeMessage: '输入您的消息...',
      poweredBy: '技术支持',
      you: '您',
      endChat: '结束聊天',
      endChatConfirm: '您确定要结束此聊天吗？'
    },
    ja: {
      greetingTitle: 'ようこそ',
      greetingMessage: 'こんにちは！本日はどのようなご用件でしょうか？',
      launcherText: 'チャットで相談',
      replyTime: '通常数分以内にお返事いたします',
      nameLabel: 'お名前',
      nameRequired: '*',
      namePlaceholder: 'フルネームを入力してください',
      emailLabel: 'メールアドレス',
      emailOptional: '(任意)',
      emailPlaceholder: 'フォローアップ用',
      departmentLabel: '部門',
      departmentPlaceholder: '部門を選択...',
      messageLabel: 'ご用件をお聞かせください',
      messagePlaceholder: 'ご質問や問題を説明してください...',
      startChat: 'チャットを開始',
      startingChat: '開始中...',
      sendMessage: '送信',
      typeMessage: 'メッセージを入力...',
      poweredBy: 'Powered by',
      you: 'あなた',
      endChat: 'チャットを終了',
      endChatConfirm: 'このチャットを終了してもよろしいですか？'
    },
    ko: {
      greetingTitle: '환영합니다',
      greetingMessage: '안녕하세요! 무엇을 도와드릴까요?',
      launcherText: '채팅 상담',
      replyTime: '보통 몇 분 내에 답변드립니다',
      nameLabel: '이름',
      nameRequired: '*',
      namePlaceholder: '성함을 입력해주세요',
      emailLabel: '이메일',
      emailOptional: '(선택사항)',
      emailPlaceholder: '후속 연락용',
      departmentLabel: '부서',
      departmentPlaceholder: '부서를 선택하세요...',
      messageLabel: '무엇을 도와드릴까요?',
      messagePlaceholder: '질문이나 문제를 설명해주세요...',
      startChat: '채팅 시작',
      startingChat: '시작 중...',
      sendMessage: '메시지 보내기',
      typeMessage: '메시지를 입력하세요...',
      poweredBy: 'Powered by',
      you: '나',
      endChat: '채팅 종료',
      endChatConfirm: '이 채팅을 종료하시겠습니까?'
    },
    ar: {
      greetingTitle: 'مرحباً',
      greetingMessage: 'أهلاً! كيف يمكننا مساعدتك اليوم؟',
      launcherText: 'تحدث معنا',
      replyTime: 'نرد عادة في غضون دقائق قليلة',
      nameLabel: 'الاسم',
      nameRequired: '*',
      namePlaceholder: 'أدخل اسمك الكامل',
      emailLabel: 'البريد الإلكتروني',
      emailOptional: '(اختياري)',
      emailPlaceholder: 'للمتابعة',
      departmentLabel: 'القسم',
      departmentPlaceholder: 'اختر قسماً...',
      messageLabel: 'كيف يمكننا المساعدة؟',
      messagePlaceholder: 'صف سؤالك أو مشكلتك...',
      startChat: 'بدء المحادثة',
      startingChat: 'جاري البدء...',
      sendMessage: 'إرسال',
      typeMessage: 'اكتب رسالتك...',
      poweredBy: 'مدعوم من',
      you: 'أنت',
      endChat: 'إنهاء المحادثة',
      endChatConfirm: 'هل أنت متأكد أنك تريد إنهاء هذه المحادثة؟'
    },
    hi: {
      greetingTitle: 'स्वागत है',
      greetingMessage: 'नमस्ते! आज हम आपकी कैसे मदद कर सकते हैं?',
      launcherText: 'चैट करें',
      replyTime: 'हम आमतौर पर कुछ मिनटों में जवाब देते हैं',
      nameLabel: 'नाम',
      nameRequired: '*',
      namePlaceholder: 'अपना पूरा नाम दर्ज करें',
      emailLabel: 'ईमेल',
      emailOptional: '(वैकल्पिक)',
      emailPlaceholder: 'फॉलो-अप के लिए',
      departmentLabel: 'विभाग',
      departmentPlaceholder: 'विभाग चुनें...',
      messageLabel: 'हम कैसे मदद कर सकते हैं?',
      messagePlaceholder: 'अपना प्रश्न या समस्या बताएं...',
      startChat: 'चैट शुरू करें',
      startingChat: 'शुरू हो रहा है...',
      sendMessage: 'भेजें',
      typeMessage: 'अपना संदेश लिखें...',
      poweredBy: 'द्वारा संचालित',
      you: 'आप',
      endChat: 'चैट समाप्त करें',
      endChatConfirm: 'क्या आप वाकई इस चैट को समाप्त करना चाहते हैं?'
    },
    nl: {
      greetingTitle: 'Welkom',
      greetingMessage: 'Hallo! Hoe kunnen we u vandaag helpen?',
      launcherText: 'Chat met ons',
      replyTime: 'We reageren meestal binnen enkele minuten',
      nameLabel: 'Naam',
      nameRequired: '*',
      namePlaceholder: 'Voer uw volledige naam in',
      emailLabel: 'E-mail',
      emailOptional: '(optioneel)',
      emailPlaceholder: 'Voor opvolging',
      departmentLabel: 'Afdeling',
      departmentPlaceholder: 'Selecteer een afdeling...',
      messageLabel: 'Hoe kunnen we helpen?',
      messagePlaceholder: 'Beschrijf uw vraag of probleem...',
      startChat: 'Start Chat',
      startingChat: 'Chat starten...',
      sendMessage: 'Verstuur bericht',
      typeMessage: 'Typ uw bericht...',
      poweredBy: 'Mogelijk gemaakt door',
      you: 'U',
      endChat: 'Chat beëindigen',
      endChatConfirm: 'Weet u zeker dat u deze chat wilt beëindigen?'
    },
    pl: {
      greetingTitle: 'Witamy',
      greetingMessage: 'Cześć! Jak możemy Ci dzisiaj pomóc?',
      launcherText: 'Porozmawiaj z nami',
      replyTime: 'Zwykle odpowiadamy w ciągu kilku minut',
      nameLabel: 'Imię',
      nameRequired: '*',
      namePlaceholder: 'Wpisz swoje pełne imię',
      emailLabel: 'E-mail',
      emailOptional: '(opcjonalnie)',
      emailPlaceholder: 'Do kontaktu zwrotnego',
      departmentLabel: 'Dział',
      departmentPlaceholder: 'Wybierz dział...',
      messageLabel: 'Jak możemy pomóc?',
      messagePlaceholder: 'Opisz swoje pytanie lub problem...',
      startChat: 'Rozpocznij Chat',
      startingChat: 'Uruchamianie...',
      sendMessage: 'Wyślij wiadomość',
      typeMessage: 'Wpisz wiadomość...',
      poweredBy: 'Obsługiwane przez',
      you: 'Ty',
      endChat: 'Zakończ czat',
      endChatConfirm: 'Czy na pewno chcesz zakończyć ten czat?'
    },
    tr: {
      greetingTitle: 'Hoş Geldiniz',
      greetingMessage: 'Merhaba! Size bugün nasıl yardımcı olabiliriz?',
      launcherText: 'Bizimle sohbet edin',
      replyTime: 'Genellikle birkaç dakika içinde yanıt veririz',
      nameLabel: 'İsim',
      nameRequired: '*',
      namePlaceholder: 'Tam adınızı girin',
      emailLabel: 'E-posta',
      emailOptional: '(isteğe bağlı)',
      emailPlaceholder: 'Takip iletişimi için',
      departmentLabel: 'Departman',
      departmentPlaceholder: 'Bir departman seçin...',
      messageLabel: 'Size nasıl yardımcı olabiliriz?',
      messagePlaceholder: 'Sorunuzu veya sorununuzu açıklayın...',
      startChat: 'Sohbeti Başlat',
      startingChat: 'Başlatılıyor...',
      sendMessage: 'Mesaj gönder',
      typeMessage: 'Mesajınızı yazın...',
      poweredBy: 'Tarafından desteklenmektedir',
      you: 'Siz',
      endChat: 'Sohbeti bitir',
      endChatConfirm: 'Bu sohbeti bitirmek istediğinizden emin misiniz?'
    },
    th: {
      greetingTitle: 'ยินดีต้อนรับ',
      greetingMessage: 'สวัสดี! เราช่วยอะไรคุณได้บ้างวันนี้?',
      launcherText: 'แชทกับเรา',
      replyTime: 'เรามักจะตอบกลับภายในไม่กี่นาที',
      nameLabel: 'ชื่อ',
      nameRequired: '*',
      namePlaceholder: 'กรอกชื่อเต็มของคุณ',
      emailLabel: 'อีเมล',
      emailOptional: '(ไม่บังคับ)',
      emailPlaceholder: 'สำหรับการติดต่อกลับ',
      departmentLabel: 'แผนก',
      departmentPlaceholder: 'เลือกแผนก...',
      messageLabel: 'เราช่วยอะไรได้บ้าง?',
      messagePlaceholder: 'อธิบายคำถามหรือปัญหาของคุณ...',
      startChat: 'เริ่มแชท',
      startingChat: 'กำลังเริ่ม...',
      sendMessage: 'ส่งข้อความ',
      typeMessage: 'พิมพ์ข้อความของคุณ...',
      poweredBy: 'ขับเคลื่อนโดย',
      you: 'คุณ',
      endChat: 'สิ้นสุดแชท',
      endChatConfirm: 'คุณแน่ใจหรือไม่ว่าต้องการสิ้นสุดแชทนี้?'
    },
    vi: {
      greetingTitle: 'Chào mừng',
      greetingMessage: 'Xin chào! Chúng tôi có thể giúp gì cho bạn?',
      launcherText: 'Chat với chúng tôi',
      replyTime: 'Chúng tôi thường phản hồi trong vài phút',
      nameLabel: 'Tên',
      nameRequired: '*',
      namePlaceholder: 'Nhập họ tên đầy đủ',
      emailLabel: 'Email',
      emailOptional: '(tùy chọn)',
      emailPlaceholder: 'Để liên hệ lại',
      departmentLabel: 'Phòng ban',
      departmentPlaceholder: 'Chọn phòng ban...',
      messageLabel: 'Chúng tôi có thể giúp gì?',
      messagePlaceholder: 'Mô tả câu hỏi hoặc vấn đề của bạn...',
      startChat: 'Bắt đầu Chat',
      startingChat: 'Đang bắt đầu...',
      sendMessage: 'Gửi tin nhắn',
      typeMessage: 'Nhập tin nhắn...',
      poweredBy: 'Được hỗ trợ bởi',
      you: 'Bạn',
      endChat: 'Kết thúc chat',
      endChatConfirm: 'Bạn có chắc chắn muốn kết thúc cuộc trò chuyện này không?'
    },
    uk: {
      greetingTitle: 'Ласкаво просимо',
      greetingMessage: 'Привіт! Чим ми можемо вам допомогти?',
      launcherText: 'Написати нам',
      replyTime: 'Зазвичай відповідаємо протягом кількох хвилин',
      nameLabel: "Ім'я",
      nameRequired: '*',
      namePlaceholder: "Введіть ваше повне ім'я",
      emailLabel: 'Ел. пошта',
      emailOptional: "(необов'язково)",
      emailPlaceholder: "Для зворотного зв'язку",
      departmentLabel: 'Відділ',
      departmentPlaceholder: 'Виберіть відділ...',
      messageLabel: 'Чим ми можемо допомогти?',
      messagePlaceholder: 'Опишіть ваше питання або проблему...',
      startChat: 'Почати чат',
      startingChat: 'Запуск...',
      sendMessage: 'Надіслати',
      typeMessage: 'Введіть повідомлення...',
      poweredBy: 'Працює на',
      you: 'Ви',
      endChat: 'Завершити чат',
      endChatConfirm: 'Ви впевнені, що хочете завершити цей чат?'
    },
    sv: {
      greetingTitle: 'Välkommen',
      greetingMessage: 'Hej! Hur kan vi hjälpa dig idag?',
      launcherText: 'Chatta med oss',
      replyTime: 'Vi svarar vanligtvis inom några minuter',
      nameLabel: 'Namn',
      nameRequired: '*',
      namePlaceholder: 'Ange ditt fullständiga namn',
      emailLabel: 'E-post',
      emailOptional: '(valfritt)',
      emailPlaceholder: 'För uppföljning',
      departmentLabel: 'Avdelning',
      departmentPlaceholder: 'Välj en avdelning...',
      messageLabel: 'Hur kan vi hjälpa?',
      messagePlaceholder: 'Beskriv din fråga eller ditt problem...',
      startChat: 'Starta Chatt',
      startingChat: 'Startar...',
      sendMessage: 'Skicka meddelande',
      typeMessage: 'Skriv ditt meddelande...',
      poweredBy: 'Drivs av',
      you: 'Du',
      endChat: 'Avsluta chatt',
      endChatConfirm: 'Är du säker på att du vill avsluta denna chatt?'
    },
    cs: {
      greetingTitle: 'Vítejte',
      greetingMessage: 'Ahoj! Jak vám dnes můžeme pomoci?',
      launcherText: 'Napište nám',
      replyTime: 'Obvykle odpovídáme během několika minut',
      nameLabel: 'Jméno',
      nameRequired: '*',
      namePlaceholder: 'Zadejte své celé jméno',
      emailLabel: 'E-mail',
      emailOptional: '(volitelné)',
      emailPlaceholder: 'Pro následnou komunikaci',
      departmentLabel: 'Oddělení',
      departmentPlaceholder: 'Vyberte oddělení...',
      messageLabel: 'Jak vám můžeme pomoci?',
      messagePlaceholder: 'Popište svůj dotaz nebo problém...',
      startChat: 'Zahájit Chat',
      startingChat: 'Spouštění...',
      sendMessage: 'Odeslat zprávu',
      typeMessage: 'Napište zprávu...',
      poweredBy: 'Poháněno',
      you: 'Vy',
      endChat: 'Ukončit chat',
      endChatConfirm: 'Opravdu chcete ukončit tento chat?'
    },
    el: {
      greetingTitle: 'Καλώς ήρθατε',
      greetingMessage: 'Γεια! Πώς μπορούμε να σας βοηθήσουμε σήμερα;',
      launcherText: 'Συνομιλήστε μαζί μας',
      replyTime: 'Συνήθως απαντάμε σε λίγα λεπτά',
      nameLabel: 'Όνομα',
      nameRequired: '*',
      namePlaceholder: 'Εισάγετε το πλήρες όνομά σας',
      emailLabel: 'Email',
      emailOptional: '(προαιρετικό)',
      emailPlaceholder: 'Για επακόλουθη επικοινωνία',
      departmentLabel: 'Τμήμα',
      departmentPlaceholder: 'Επιλέξτε τμήμα...',
      messageLabel: 'Πώς μπορούμε να βοηθήσουμε;',
      messagePlaceholder: 'Περιγράψτε την ερώτηση ή το πρόβλημά σας...',
      startChat: 'Έναρξη Συνομιλίας',
      startingChat: 'Εκκίνηση...',
      sendMessage: 'Αποστολή μηνύματος',
      typeMessage: 'Πληκτρολογήστε το μήνυμά σας...',
      poweredBy: 'Με την υποστήριξη',
      you: 'Εσείς',
      endChat: 'Τερματισμός συνομιλίας',
      endChatConfirm: 'Είστε σίγουροι ότι θέλετε να τερματίσετε αυτή τη συνομιλία;'
    },
    he: {
      greetingTitle: 'ברוכים הבאים',
      greetingMessage: 'שלום! איך נוכל לעזור לך היום?',
      launcherText: 'צ\'אט איתנו',
      replyTime: 'אנחנו בדרך כלל עונים תוך מספר דקות',
      nameLabel: 'שם',
      nameRequired: '*',
      namePlaceholder: 'הזן את שמך המלא',
      emailLabel: 'אימייל',
      emailOptional: '(אופציונלי)',
      emailPlaceholder: 'לתקשורת המשך',
      departmentLabel: 'מחלקה',
      departmentPlaceholder: 'בחר מחלקה...',
      messageLabel: 'איך נוכל לעזור?',
      messagePlaceholder: 'תאר את השאלה או הבעיה שלך...',
      startChat: 'התחל צ\'אט',
      startingChat: 'מתחיל...',
      sendMessage: 'שלח הודעה',
      typeMessage: 'הקלד את ההודעה שלך...',
      poweredBy: 'מופעל על ידי',
      you: 'אתה',
      endChat: 'סיים צ\'אט',
      endChatConfirm: 'האם אתה בטוח שברצונך לסיים את הצ\'אט הזה?'
    },
    id: {
      greetingTitle: 'Selamat Datang',
      greetingMessage: 'Halo! Bagaimana kami bisa membantu Anda hari ini?',
      launcherText: 'Chat dengan kami',
      replyTime: 'Kami biasanya membalas dalam beberapa menit',
      nameLabel: 'Nama',
      nameRequired: '*',
      namePlaceholder: 'Masukkan nama lengkap Anda',
      emailLabel: 'Email',
      emailOptional: '(opsional)',
      emailPlaceholder: 'Untuk komunikasi lanjutan',
      departmentLabel: 'Departemen',
      departmentPlaceholder: 'Pilih departemen...',
      messageLabel: 'Bagaimana kami bisa membantu?',
      messagePlaceholder: 'Jelaskan pertanyaan atau masalah Anda...',
      startChat: 'Mulai Chat',
      startingChat: 'Memulai...',
      sendMessage: 'Kirim pesan',
      typeMessage: 'Ketik pesan Anda...',
      poweredBy: 'Didukung oleh',
      you: 'Anda',
      endChat: 'Akhiri chat',
      endChatConfirm: 'Apakah Anda yakin ingin mengakhiri chat ini?'
    },
    ms: {
      greetingTitle: 'Selamat Datang',
      greetingMessage: 'Hai! Bagaimana kami boleh membantu anda hari ini?',
      launcherText: 'Sembang dengan kami',
      replyTime: 'Kami biasanya membalas dalam beberapa minit',
      nameLabel: 'Nama',
      nameRequired: '*',
      namePlaceholder: 'Masukkan nama penuh anda',
      emailLabel: 'E-mel',
      emailOptional: '(pilihan)',
      emailPlaceholder: 'Untuk komunikasi susulan',
      departmentLabel: 'Jabatan',
      departmentPlaceholder: 'Pilih jabatan...',
      messageLabel: 'Bagaimana kami boleh membantu?',
      messagePlaceholder: 'Terangkan soalan atau masalah anda...',
      startChat: 'Mula Sembang',
      startingChat: 'Memulakan...',
      sendMessage: 'Hantar mesej',
      typeMessage: 'Taip mesej anda...',
      poweredBy: 'Dikuasakan oleh',
      you: 'Anda',
      endChat: 'Tamatkan sembang',
      endChatConfirm: 'Adakah anda pasti mahu menamatkan sembang ini?'
    },
    fa: {
      greetingTitle: 'خوش آمدید',
      greetingMessage: 'سلام! چگونه می‌توانیم به شما کمک کنیم؟',
      launcherText: 'گفتگو با ما',
      replyTime: 'معمولاً در عرض چند دقیقه پاسخ می‌دهیم',
      nameLabel: 'نام',
      nameRequired: '*',
      namePlaceholder: 'نام کامل خود را وارد کنید',
      emailLabel: 'ایمیل',
      emailOptional: '(اختیاری)',
      emailPlaceholder: 'برای پیگیری',
      departmentLabel: 'بخش',
      departmentPlaceholder: 'یک بخش انتخاب کنید...',
      messageLabel: 'چگونه می‌توانیم کمک کنیم؟',
      messagePlaceholder: 'سوال یا مشکل خود را توضیح دهید...',
      startChat: 'شروع گفتگو',
      startingChat: 'در حال شروع...',
      sendMessage: 'ارسال پیام',
      typeMessage: 'پیام خود را بنویسید...',
      poweredBy: 'قدرت گرفته از',
      you: 'شما',
      endChat: 'پایان گفتگو',
      endChatConfirm: 'آیا مطمئن هستید که می‌خواهید این گفتگو را به پایان برسانید؟'
    }
  };

  // Get browser language
  function detectLanguage() {
    const browserLang = navigator.language || navigator.userLanguage || 'en';
    const langCode = browserLang.split('-')[0].toLowerCase();
    return i18n[langCode] ? langCode : 'en';
  }

  // Get resolved locale (handles 'auto')
  function getLocale() {
    return config.locale === 'auto' ? detectLanguage() : (config.locale || 'en');
  }

  // Get translation
  function t(key) {
    const lang = getLocale();
    const translations = i18n[lang] || i18n.en;
    return translations[key] || i18n.en[key] || key;
  }

  // Default configuration
  const DEFAULT_CONFIG = {
    baseUrl: '',
    websiteToken: '',
    position: 'right', // 'left' or 'right'
    locale: 'auto', // 'auto' to detect, or specific locale code
    darkMode: 'light', // 'light', 'dark', or 'auto' (follows browser preference)
    zIndex: 999999,

    // Launcher customization
    launcherSize: 60,
    launcherIcon: null, // Custom SVG string, null for default

    // Colors
    brandColor: '#3B82F6',
    headerColor: null, // null = use brandColor
    textColor: '#FFFFFF',

    // Window dimensions
    windowWidth: 380,
    windowHeight: 600,
    borderRadius: 16,

    // Custom messages (override i18n)
    greetingTitle: null, // null = use i18n
    greetingMessage: null, // null = use i18n
    launcherText: null, // null = use i18n

    // Pre-fill user info
    userName: null,
    userEmail: null,
    userAttributes: {},
    conversationAttributes: {},

    // Behavior
    showAvatar: true,
    hideOnMobile: false,
    hidePoweredBy: false,

    // Notifications
    soundEnabled: true,       // Play sound on new message
    soundType: 'chime',       // 'chime', 'bell', 'ding', 'pop', 'none'
    titleNotification: true,  // Update document title with new message
    titlePrefix: '💬 ',       // Prefix for title notification

    // Auto-open
    autoOpen: 0,              // Auto-open after X ms (0 = disabled)
    autoOpenOnce: true,       // Only auto-open once per session

    // Messages
    offlineMessage: null,     // Message when agents are offline

    // Pre-chat form
    preChatFormFields: ['name', 'email'], // Fields: name, email, phone, company, message
    defaultDepartmentId: null, // Auto-assign to department

    // Transcript
    transcriptEmail: false,   // Allow users to email transcript

    // Custom styling
    customCSS: null,          // Custom CSS to inject

    // Custom field definitions for pre-chat form
    // Array of { name, title, type, required, placeholder }
    // type: 'string' | 'int' | 'float' | 'date' | 'email' | 'tel' | 'textarea'
    customFields: []
  };

  // SDK State
  let config = { ...DEFAULT_CONFIG };
  let isInitialized = false;
  let isOpen = false;
  let conversation = null;
  let messages = [];
  let websocket = null;
  let elements = {};
  let user = null;
  let customAttributes = {};
  let eventCallbacks = {};
  let reconnectAttempts = 0;
  const MAX_RECONNECT_ATTEMPTS = 5;
  const RECONNECT_DELAY = 3000;
  let originalTitle = '';
  let audioContext = null;

  // Storage keys
  const STORAGE_KEYS = {
    CONVERSATION: 'homa_chat_conversation',
    USER: 'homa_chat_user',
    MESSAGES: 'homa_chat_messages'
  };

  // ==========================================
  // Utility Functions
  // ==========================================

  // Helper to determine if dark mode should be used
  function isDarkMode() {
    if (config.darkMode === 'dark') return true;
    if (config.darkMode === 'light') return false;
    // Auto mode - check browser preference
    if (config.darkMode === 'auto') {
      return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
    }
    // Legacy support: boolean values
    return config.darkMode === true;
  }

  // Sound notification frequencies for different sound types
  const SOUND_TYPES = {
    chime: { freq: [800, 1000, 1200], duration: 0.15, type: 'sine' },
    bell: { freq: [523, 659, 784], duration: 0.2, type: 'sine' },
    ding: { freq: [1400], duration: 0.1, type: 'sine' },
    pop: { freq: [400, 600], duration: 0.08, type: 'square' },
    bubble: { freq: [600, 800], duration: 0.12, type: 'sine' },
    drop: { freq: [1200, 800, 600], duration: 0.1, type: 'sine' },
    ping: { freq: [1800], duration: 0.08, type: 'sine' },
    pluck: { freq: [400, 300], duration: 0.15, type: 'triangle' },
    tap: { freq: [800], duration: 0.05, type: 'square' },
    whoosh: { freq: [200, 400, 800], duration: 0.08, type: 'sawtooth' },
    none: null
  };

  // Play notification sound using Web Audio API
  function playNotificationSound() {
    if (!config.soundEnabled || config.soundType === 'none') return;

    try {
      if (!audioContext) {
        audioContext = new (window.AudioContext || window.webkitAudioContext)();
      }

      const soundConfig = SOUND_TYPES[config.soundType] || SOUND_TYPES.chime;
      if (!soundConfig) return;

      const now = audioContext.currentTime;

      soundConfig.freq.forEach((freq, i) => {
        const oscillator = audioContext.createOscillator();
        const gainNode = audioContext.createGain();

        oscillator.connect(gainNode);
        gainNode.connect(audioContext.destination);

        oscillator.frequency.value = freq;
        oscillator.type = soundConfig.type;

        gainNode.gain.setValueAtTime(0.3, now + i * soundConfig.duration);
        gainNode.gain.exponentialRampToValueAtTime(0.01, now + (i + 1) * soundConfig.duration);

        oscillator.start(now + i * soundConfig.duration);
        oscillator.stop(now + (i + 1) * soundConfig.duration);
      });
    } catch (e) {
      console.warn('Homa Chat: Could not play notification sound', e);
    }
  }

  // Update document title with new message notification
  function updateTitleNotification(message) {
    if (!config.titleNotification) return;

    // Store original title on first notification
    if (!originalTitle) {
      originalTitle = document.title;
    }

    // Truncate message to 30 chars
    const truncated = message.length > 30 ? message.substring(0, 30) + '...' : message;
    document.title = config.titlePrefix + truncated;
  }

  // Restore original document title
  function restoreTitle() {
    if (originalTitle) {
      document.title = originalTitle;
    }
  }

  // Handle new message notification
  function notifyNewMessage(message) {
    // Only notify for agent messages when widget is closed or tab is not focused
    if (!isOpen || document.hidden) {
      playNotificationSound();
      if (message && message.content) {
        // Strip HTML tags for title
        const textContent = message.content.replace(/<[^>]*>/g, '').trim();
        if (textContent) {
          updateTitleNotification(textContent);
        }
      }
    }
  }

  function generateUUID() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
      const r = Math.random() * 16 | 0;
      const v = c === 'x' ? r : (r & 0x3 | 0x8);
      return v.toString(16);
    });
  }

  function safeLocalStorage(action, key, value) {
    try {
      if (action === 'get') {
        const item = localStorage.getItem(key);
        return item ? JSON.parse(item) : null;
      } else if (action === 'set') {
        localStorage.setItem(key, JSON.stringify(value));
      } else if (action === 'remove') {
        localStorage.removeItem(key);
      }
    } catch (e) {
      console.warn('HomaChat: LocalStorage not available', e);
      return null;
    }
  }

  function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
      const later = () => {
        clearTimeout(timeout);
        func(...args);
      };
      clearTimeout(timeout);
      timeout = setTimeout(later, wait);
    };
  }

  function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  function formatTime(date) {
    const d = new Date(date);
    return d.toLocaleTimeString(getLocale(), { hour: '2-digit', minute: '2-digit' });
  }

  function scrollToBottom() {
    if (elements.messagesContainer) {
      // Use setTimeout to ensure DOM has updated
      setTimeout(() => {
        elements.messagesContainer.scrollTop = elements.messagesContainer.scrollHeight;
      }, 0);
    }
  }

  function emit(eventName, data) {
    if (eventCallbacks[eventName]) {
      eventCallbacks[eventName].forEach(callback => {
        try {
          callback(data);
        } catch (e) {
          console.error('HomaChat: Event callback error', e);
        }
      });
    }
  }

  // ==========================================
  // API Functions
  // ==========================================

  async function apiRequest(endpoint, method = 'GET', body = null) {
    const url = `${config.baseUrl}${endpoint}`;
    const headers = {
      'Content-Type': 'application/json'
    };

    const options = {
      method,
      headers,
      credentials: 'omit'  // Don't send cookies - we use URL-based secret auth
    };

    if (body) {
      options.body = JSON.stringify(body);
    }

    try {
      const response = await fetch(url, options);
      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || 'Request failed');
      }

      return data;
    } catch (error) {
      console.error('HomaChat API Error:', error);
      throw error;
    }
  }

  async function createConversation() {
    const clientName = user?.name || 'Website Visitor';
    const clientEmail = user?.email || null;

    const payload = {
      title: `Chat from ${clientName}`,
      status: 'new',
      priority: 'medium',
      client_name: clientName,
      client_email: clientEmail,
      client_attributes: { ...customAttributes, ...user?.attributes },
      parameters: {
        source: 'widget',
        page_url: window.location.href,
        referrer: document.referrer,
        user_agent: navigator.userAgent
      }
    };

    try {
      const response = await apiRequest('/api/client/conversations', 'PUT', payload);
      conversation = {
        id: response.data.id,
        secret: response.data.secret
      };

      // Store conversation
      safeLocalStorage('set', STORAGE_KEYS.CONVERSATION, conversation);

      // Connect WebSocket
      connectWebSocket();

      emit('conversation:created', conversation);

      return conversation;
    } catch (error) {
      emit('error', { type: 'conversation_create', error });
      throw error;
    }
  }

  async function sendMessage(content) {
    if (!conversation) {
      await createConversation();
    }

    const endpoint = `/api/client/conversations/${conversation.id}/${conversation.secret}/messages`;

    try {
      const response = await apiRequest(endpoint, 'POST', { message: content });

      const message = {
        id: response.data?.id || generateUUID(),
        body: content,
        is_client: true,
        user_name: user?.name || t('you'),
        user_avatar: null,
        created_at: new Date().toISOString()
      };

      messages.push(message);
      renderMessages();
      safeLocalStorage('set', STORAGE_KEYS.MESSAGES, messages);

      emit('message:sent', message);

      return message;
    } catch (error) {
      emit('error', { type: 'message_send', error });
      throw error;
    }
  }

  async function loadMessages() {
    if (!conversation) return;

    const endpoint = `/api/client/conversations/${conversation.id}/${conversation.secret}`;

    try {
      const response = await apiRequest(endpoint);
      const rawMessages = response.data?.messages || [];
      const conversationData = response.data?.conversation;

      // Normalize messages to include user info
      messages = rawMessages.map(msg => {
        const isClient = !!msg.client_id && !msg.user_id;

        // Get agent name - prefer display_name, then combine name + last_name
        let agentName = 'Support Agent';
        if (!isClient && msg.user) {
          if (msg.user.display_name && msg.user.display_name.trim()) {
            agentName = msg.user.display_name;
          } else if (msg.user.name) {
            agentName = msg.user.name + (msg.user.last_name ? ' ' + msg.user.last_name : '');
          }
        }

        return {
          id: msg.id,
          body: msg.body,
          is_client: isClient,
          user_name: agentName,
          user_avatar: msg.user?.avatar || null,
          created_at: msg.created_at
        };
      });

      safeLocalStorage('set', STORAGE_KEYS.MESSAGES, messages);
      renderMessages();

      // Auto-open widget if conversation is not closed
      if (conversationData && conversationData.status !== 'closed' && !isOpen) {
        openWidget();
      }
    } catch (error) {
      console.error('HomaChat: Failed to load messages', error);
    }
  }

  // ==========================================
  // WebSocket Functions
  // ==========================================

  function connectWebSocket() {
    if (!conversation || websocket) return;

    const protocol = config.baseUrl.startsWith('https') ? 'wss' : 'ws';
    const host = config.baseUrl.replace(/^https?:\/\//, '');
    const wsUrl = `${protocol}://${host}/ws/conversations/${conversation.id}/${conversation.secret}`;

    try {
      websocket = new WebSocket(wsUrl);

      websocket.onopen = () => {
        console.log('HomaChat: WebSocket connected');
        reconnectAttempts = 0;
        emit('websocket:connected');
      };

      websocket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          handleWebSocketMessage(data);
        } catch (e) {
          console.error('HomaChat: Failed to parse WebSocket message', e);
        }
      };

      websocket.onerror = (error) => {
        console.error('HomaChat: WebSocket error', error);
        emit('websocket:error', error);
      };

      websocket.onclose = () => {
        console.log('HomaChat: WebSocket closed');
        websocket = null;
        emit('websocket:disconnected');

        // Attempt to reconnect
        if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
          reconnectAttempts++;
          setTimeout(connectWebSocket, RECONNECT_DELAY * reconnectAttempts);
        }
      };
    } catch (error) {
      console.error('HomaChat: Failed to create WebSocket', error);
    }
  }

  function handleWebSocketMessage(data) {
    console.log('HomaChat: WebSocket message received', data);
    // Handle different message types
    if (data.type === 'message.created' || data.event === 'message.created' || data.event === 'message_created') {
      const message = data.message || data.data;
      console.log('HomaChat: Message event detected', { message, sender_type: data.sender_type });

      // Check if message is from agent (sender_type === 'agent' OR message has user_id but not client_id)
      const isAgentMessage = data.sender_type === 'agent' ||
                             (message && message.user_id && !message.client_id);
      console.log('HomaChat: isAgentMessage =', isAgentMessage);

      if (message && isAgentMessage) {
        // Get agent name - prefer display_name, then combine name + last_name, fallback to 'Support Agent'
        let agentName = 'Support Agent';
        if (message.user) {
          if (message.user.display_name && message.user.display_name.trim()) {
            agentName = message.user.display_name;
          } else if (message.user.name) {
            agentName = message.user.name + (message.user.last_name ? ' ' + message.user.last_name : '');
          }
        }

        messages.push({
          id: message.id,
          body: message.body,
          is_client: false,
          user_name: agentName,
          user_avatar: message.user?.avatar || null,
          created_at: message.created_at
        });
        renderMessages();
        safeLocalStorage('set', STORAGE_KEYS.MESSAGES, messages);
        emit('message:received', message);

        // Play sound and update title for new agent message
        notifyNewMessage({ content: message.body });

        // Show notification if widget is closed
        if (!isOpen) {
          showNotification();
        }
      }
    } else if (data.type === 'typing' || data.event === 'typing') {
      showTypingIndicator(data.is_typing);
    }
  }

  function disconnectWebSocket() {
    if (websocket) {
      websocket.close();
      websocket = null;
    }
  }

  // ==========================================
  // UI Rendering
  // ==========================================

  function createStyles() {
    // Calculate derived values
    const headerColor = config.headerColor || config.brandColor;
    const iconSize = Math.round(config.launcherSize * 0.47);
    const isRTL = ['ar', 'he', 'fa'].includes(getLocale());

    const style = document.createElement('style');
    style.id = 'homa-chat-styles';
    style.textContent = `
      .homa-chat-widget * {
        box-sizing: border-box;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
        margin: 0;
        padding: 0;
      }

      .homa-chat-widget {
        position: fixed;
        bottom: 20px;
        ${config.position}: 20px;
        z-index: ${config.zIndex};
        font-size: 14px;
        direction: ${isRTL ? 'rtl' : 'ltr'};
      }

      .homa-chat-launcher {
        width: ${config.launcherSize}px;
        height: ${config.launcherSize}px;
        border-radius: 50%;
        background-color: ${config.brandColor};
        border: none;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        transition: transform 0.2s, box-shadow 0.2s;
      }

      .homa-chat-launcher:hover {
        transform: scale(1.1);
        box-shadow: 0 6px 20px rgba(0, 0, 0, 0.2);
      }

      .homa-chat-launcher svg {
        width: ${iconSize}px;
        height: ${iconSize}px;
        fill: ${config.textColor};
      }

      .homa-chat-launcher-badge {
        position: absolute;
        top: -5px;
        ${isRTL ? 'left' : 'right'}: -5px;
        background-color: #EF4444;
        color: white;
        font-size: 12px;
        font-weight: 600;
        min-width: 20px;
        height: 20px;
        border-radius: 10px;
        display: none;
        align-items: center;
        justify-content: center;
        padding: 0 6px;
      }

      .homa-chat-window {
        position: absolute;
        bottom: ${config.launcherSize + 20}px;
        ${config.position}: 0;
        width: ${config.windowWidth}px;
        height: ${config.windowHeight}px;
        max-height: calc(100vh - 120px);
        background-color: ${isDarkMode() ? '#1F2937' : '#FFFFFF'};
        border-radius: ${config.borderRadius}px;
        box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
        display: none;
        flex-direction: column;
        overflow: hidden;
        animation: homa-slide-up 0.3s ease;
      }

      @keyframes homa-slide-up {
        from {
          opacity: 0;
          transform: translateY(20px);
        }
        to {
          opacity: 1;
          transform: translateY(0);
        }
      }

      .homa-chat-window.open {
        display: flex;
      }

      .homa-chat-header {
        background-color: ${headerColor};
        color: ${config.textColor};
        padding: 14px 16px;
        display: flex;
        align-items: center;
        justify-content: space-between;
        border-radius: ${config.borderRadius}px ${config.borderRadius}px 0 0;
        flex-shrink: 0;
      }

      .homa-chat-header-info {
        display: flex;
        align-items: center;
        gap: 12px;
      }

      .homa-chat-header-avatar {
        width: 40px;
        height: 40px;
        border-radius: 50%;
        background-color: rgba(255, 255, 255, 0.2);
        display: flex;
        align-items: center;
        justify-content: center;
      }

      .homa-chat-header-avatar svg {
        width: 24px;
        height: 24px;
        fill: ${config.textColor};
      }

      .homa-chat-header-text h4 {
        font-size: 16px;
        font-weight: 600;
        margin-bottom: 2px;
      }

      .homa-chat-header-text p {
        font-size: 12px;
        opacity: 0.9;
      }

      .homa-chat-header-buttons {
        display: flex;
        align-items: center;
        gap: 4px;
      }

      .homa-chat-minimize,
      .homa-chat-close {
        background: none;
        border: none;
        cursor: pointer;
        padding: 8px;
        border-radius: 8px;
        transition: background-color 0.2s;
        display: flex;
        align-items: center;
        justify-content: center;
      }

      .homa-chat-minimize:hover {
        background-color: rgba(255, 255, 255, 0.1);
      }

      .homa-chat-close:hover {
        background-color: rgba(239, 68, 68, 0.2);
      }

      .homa-chat-minimize svg,
      .homa-chat-close svg {
        width: 20px;
        height: 20px;
        fill: ${config.textColor};
      }

      .homa-chat-close:hover svg {
        fill: #EF4444;
      }

      .homa-chat-messages {
        flex: 1;
        overflow-y: auto;
        padding: 20px;
        display: flex;
        flex-direction: column;
        gap: 12px;
        background-color: ${isDarkMode() ? '#111827' : '#F9FAFB'};
      }

      .homa-chat-greeting {
        text-align: center;
        padding: 20px;
        color: ${isDarkMode() ? '#9CA3AF' : '#6B7280'};
      }

      .homa-chat-greeting h5 {
        font-size: 18px;
        font-weight: 600;
        margin-bottom: 8px;
        color: ${isDarkMode() ? '#F9FAFB' : '#111827'};
      }

      .homa-chat-message-wrapper {
        display: flex;
        flex-direction: column;
        max-width: 85%;
        margin-bottom: 4px;
      }

      .homa-chat-message-wrapper.client {
        align-self: flex-end;
        align-items: flex-end;
      }

      .homa-chat-message-wrapper.agent {
        align-self: flex-start;
        align-items: flex-start;
      }

      .homa-chat-message-header {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 4px;
        padding: 0 4px;
      }

      .homa-chat-message-wrapper.client .homa-chat-message-header {
        flex-direction: row-reverse;
      }

      .homa-chat-avatar {
        width: 24px;
        height: 24px;
        border-radius: 50%;
        background-size: cover;
        background-position: center;
        background-color: ${config.brandColor};
        flex-shrink: 0;
      }

      .homa-chat-avatar.initials {
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 10px;
        font-weight: 600;
        color: ${config.textColor};
      }

      .homa-chat-message-wrapper.agent .homa-chat-avatar {
        background-color: ${isDarkMode() ? '#4B5563' : '#E5E7EB'};
        color: ${isDarkMode() ? '#F9FAFB' : '#374151'};
      }

      .homa-chat-sender-name {
        font-size: 12px;
        font-weight: 500;
        color: ${isDarkMode() ? '#9CA3AF' : '#6B7280'};
      }

      .homa-chat-message-time {
        font-size: 11px;
        color: ${isDarkMode() ? '#6B7280' : '#9CA3AF'};
      }

      .homa-chat-message {
        max-width: 80%;
        padding: 10px 14px;
        border-radius: 16px;
        line-height: 1.5;
        word-wrap: break-word;
        font-size: 14px;
      }

      .homa-chat-message.client {
        align-self: flex-end;
        background-color: ${config.brandColor};
        color: ${config.textColor};
        border-bottom-right-radius: 4px;
      }

      .homa-chat-message.agent {
        background-color: ${isDarkMode() ? '#374151' : '#FFFFFF'};
        color: ${isDarkMode() ? '#F9FAFB' : '#111827'};
        border-bottom-left-radius: 4px;
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
      }

      .homa-chat-typing {
        display: none;
        align-self: flex-start;
        padding: 12px 16px;
        background-color: ${isDarkMode() ? '#374151' : '#FFFFFF'};
        border-radius: 16px;
        border-bottom-left-radius: 4px;
      }

      .homa-chat-typing.visible {
        display: block;
      }

      .homa-chat-typing-dots {
        display: flex;
        gap: 4px;
      }

      .homa-chat-typing-dots span {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        background-color: ${isDarkMode() ? '#9CA3AF' : '#6B7280'};
        animation: homa-bounce 1.4s infinite both;
      }

      .homa-chat-typing-dots span:nth-child(2) {
        animation-delay: 0.2s;
      }

      .homa-chat-typing-dots span:nth-child(3) {
        animation-delay: 0.4s;
      }

      @keyframes homa-bounce {
        0%, 80%, 100% { transform: translateY(0); }
        40% { transform: translateY(-6px); }
      }

      .homa-chat-input-container {
        padding: 16px 20px;
        background-color: ${isDarkMode() ? '#1F2937' : '#FFFFFF'};
        border-top: 1px solid ${isDarkMode() ? '#374151' : '#E5E7EB'};
        display: flex;
        gap: 12px;
        align-items: flex-end;
      }

      .homa-chat-input {
        flex: 1;
        padding: 12px 16px;
        border: 1px solid ${isDarkMode() ? '#374151' : '#E5E7EB'};
        border-radius: 24px;
        font-size: 14px;
        resize: none;
        outline: none;
        max-height: 120px;
        background-color: ${isDarkMode() ? '#374151' : '#F9FAFB'};
        color: ${isDarkMode() ? '#F9FAFB' : '#111827'};
        transition: border-color 0.2s;
        overflow-y: auto;
        scrollbar-width: none; /* Firefox */
        -ms-overflow-style: none; /* IE/Edge */
      }

      .homa-chat-input::-webkit-scrollbar {
        display: none; /* Chrome/Safari/Opera */
      }

      .homa-chat-input:focus {
        border-color: ${config.brandColor};
      }

      .homa-chat-input::placeholder {
        color: ${isDarkMode() ? '#9CA3AF' : '#9CA3AF'};
      }

      .homa-chat-send {
        width: 44px;
        height: 44px;
        border-radius: 50%;
        background-color: ${config.brandColor};
        border: none;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: transform 0.2s, background-color 0.2s;
        flex-shrink: 0;
      }

      .homa-chat-send:hover {
        transform: scale(1.05);
      }

      .homa-chat-send:disabled {
        background-color: ${isDarkMode() ? '#4B5563' : '#D1D5DB'};
        cursor: not-allowed;
        transform: none;
      }

      .homa-chat-send svg {
        width: 20px;
        height: 20px;
        fill: ${config.textColor};
      }

      .homa-chat-powered {
        text-align: center;
        padding: 8px;
        font-size: 11px;
        color: ${isDarkMode() ? '#6B7280' : '#9CA3AF'};
        background-color: ${isDarkMode() ? '#1F2937' : '#FFFFFF'};
      }

      .homa-chat-powered a {
        color: ${config.brandColor};
        text-decoration: none;
      }

      /* Pre-chat Form Styles - Compact layout */
      .homa-chat-prechat-form {
        flex: 1;
        overflow-y: auto;
        padding: 16px;
        background-color: ${isDarkMode() ? '#111827' : '#F9FAFB'};
      }

      .homa-chat-prechat-intro {
        text-align: center;
        margin-bottom: 12px;
      }

      .homa-chat-prechat-intro h5 {
        font-size: 16px;
        font-weight: 600;
        margin-bottom: 4px;
        color: ${isDarkMode() ? '#F9FAFB' : '#111827'};
      }

      .homa-chat-prechat-intro p {
        color: ${isDarkMode() ? '#9CA3AF' : '#6B7280'};
        font-size: 13px;
        line-height: 1.4;
      }

      .homa-chat-form {
        display: flex;
        flex-direction: column;
        gap: 12px;
      }

      .homa-chat-form-group {
        display: flex;
        flex-direction: column;
        gap: 4px;
      }

      .homa-chat-form-group label {
        font-size: 12px;
        font-weight: 500;
        color: ${isDarkMode() ? '#D1D5DB' : '#374151'};
      }

      .homa-chat-form-group label .required {
        color: #EF4444;
        font-weight: 400;
      }

      .homa-chat-form-group label .optional {
        color: ${isDarkMode() ? '#6B7280' : '#9CA3AF'};
        font-weight: 400;
        font-size: 11px;
      }

      .homa-chat-form-input,
      .homa-chat-form-select,
      .homa-chat-form-textarea {
        padding: 8px 12px;
        border: 1px solid ${isDarkMode() ? '#374151' : '#E5E7EB'};
        border-radius: 6px;
        font-size: 13px;
        background-color: ${isDarkMode() ? '#374151' : '#FFFFFF'};
        color: ${isDarkMode() ? '#F9FAFB' : '#111827'};
        outline: none;
        transition: border-color 0.2s, box-shadow 0.2s;
      }

      .homa-chat-form-input:focus,
      .homa-chat-form-select:focus,
      .homa-chat-form-textarea:focus {
        border-color: ${config.brandColor};
        box-shadow: 0 0 0 2px ${config.brandColor}20;
      }

      .homa-chat-form-input::placeholder,
      .homa-chat-form-textarea::placeholder {
        color: ${isDarkMode() ? '#6B7280' : '#9CA3AF'};
      }

      .homa-chat-form-select {
        cursor: pointer;
        appearance: none;
        background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='${isDarkMode() ? '%239CA3AF' : '%236B7280'}' d='M6 8L1 3h10z'/%3E%3C/svg%3E");
        background-repeat: no-repeat;
        background-position: right 10px center;
        padding-right: 32px;
      }

      .homa-chat-form-textarea {
        resize: none;
        min-height: 60px;
        scrollbar-width: none; /* Firefox */
        -ms-overflow-style: none; /* IE/Edge */
      }

      .homa-chat-form-textarea::-webkit-scrollbar {
        display: none; /* Chrome/Safari/Opera */
      }

      .homa-chat-form-submit {
        background-color: ${config.brandColor};
        color: ${config.textColor};
        border: none;
        padding: 10px 16px;
        border-radius: 6px;
        font-size: 14px;
        font-weight: 600;
        cursor: pointer;
        transition: transform 0.2s, box-shadow 0.2s, opacity 0.2s;
        margin-top: 4px;
      }

      .homa-chat-form-submit:hover {
        transform: translateY(-1px);
        box-shadow: 0 4px 12px ${config.brandColor}40;
      }

      .homa-chat-form-submit:active {
        transform: translateY(0);
      }

      .homa-chat-form-submit:disabled {
        opacity: 0.6;
        cursor: not-allowed;
        transform: none;
        box-shadow: none;
      }

      .homa-chat-powered.hidden {
        display: none;
      }

      @media (max-width: 480px) {
        .homa-chat-widget {
          ${config.hideOnMobile ? 'display: none;' : ''}
        }

        .homa-chat-window {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          width: 100%;
          height: 100%;
          max-height: 100%;
          border-radius: 0;
        }
      }
    `;
    document.head.appendChild(style);
  }

  // Build pre-chat form fields based on config.preChatFormFields
  function buildPreChatFormFields() {
    const fields = config.preChatFormFields || ['name', 'email'];
    const customFields = config.customFields || [];
    let html = '';

    // Helper to get custom field definition
    const getCustomField = (name) => customFields.find(f => f.name === name);

    // Helper to get input type from data type
    const getInputType = (dataType) => {
      switch (dataType) {
        case 'int': return 'number';
        case 'float': return 'number';
        case 'date': return 'date';
        case 'email': return 'email';
        case 'tel': return 'tel';
        default: return 'text';
      }
    };

    fields.forEach(fieldName => {
      // Check built-in fields first
      switch (fieldName) {
        case 'name':
          html += `
            <div class="homa-chat-form-group">
              <label for="homa-prechat-name">${escapeHtml(t('nameLabel'))} <span class="required">${t('nameRequired')}</span></label>
              <input type="text" id="homa-prechat-name" class="homa-chat-form-input" placeholder="${escapeHtml(t('namePlaceholder'))}" value="${escapeHtml(config.userName || '')}" required />
            </div>`;
          break;
        case 'email':
          html += `
            <div class="homa-chat-form-group">
              <label for="homa-prechat-email">${escapeHtml(t('emailLabel'))} <span class="optional">${escapeHtml(t('emailOptional'))}</span></label>
              <input type="email" id="homa-prechat-email" class="homa-chat-form-input" placeholder="${escapeHtml(t('emailPlaceholder'))}" value="${escapeHtml(config.userEmail || '')}" />
            </div>`;
          break;
        case 'message':
          html += `
            <div class="homa-chat-form-group">
              <label for="homa-prechat-message">${escapeHtml(t('messageLabel'))} <span class="required">${t('nameRequired')}</span></label>
              <textarea id="homa-prechat-message" class="homa-chat-form-textarea" placeholder="${escapeHtml(t('messagePlaceholder'))}" required rows="2"></textarea>
            </div>`;
          break;
        default:
          // Check if it's a custom field
          const customField = getCustomField(fieldName);
          if (customField) {
            const inputType = getInputType(customField.type || customField.data_type);
            const isRequired = customField.required;
            const requiredSpan = isRequired
              ? `<span class="required">${t('nameRequired')}</span>`
              : `<span class="optional">${escapeHtml(t('emailOptional'))}</span>`;
            const requiredAttr = isRequired ? 'required' : '';
            const placeholder = customField.placeholder || '';
            const step = customField.type === 'float' || customField.data_type === 'float' ? 'step="0.01"' : '';

            if (customField.type === 'textarea') {
              html += `
                <div class="homa-chat-form-group" data-custom-field="${escapeHtml(fieldName)}">
                  <label for="homa-prechat-${escapeHtml(fieldName)}">${escapeHtml(customField.title)} ${requiredSpan}</label>
                  <textarea id="homa-prechat-${escapeHtml(fieldName)}" class="homa-chat-form-textarea" placeholder="${escapeHtml(placeholder)}" ${requiredAttr} rows="2"></textarea>
                </div>`;
            } else {
              html += `
                <div class="homa-chat-form-group" data-custom-field="${escapeHtml(fieldName)}">
                  <label for="homa-prechat-${escapeHtml(fieldName)}">${escapeHtml(customField.title)} ${requiredSpan}</label>
                  <input type="${inputType}" id="homa-prechat-${escapeHtml(fieldName)}" class="homa-chat-form-input" placeholder="${escapeHtml(placeholder)}" ${step} ${requiredAttr} />
                </div>`;
            }
          }
          break;
      }
    });

    // Always add department (will be hidden if no departments available)
    html += `
      <div class="homa-chat-form-group homa-chat-department-group">
        <label for="homa-prechat-department">${escapeHtml(t('departmentLabel'))} <span class="optional">${escapeHtml(t('emailOptional'))}</span></label>
        <select id="homa-prechat-department" class="homa-chat-form-select">
          <option value="">${escapeHtml(t('departmentPlaceholder'))}</option>
        </select>
      </div>`;

    // Add message field if not in preChatFormFields (it's always needed)
    if (!fields.includes('message')) {
      html += `
        <div class="homa-chat-form-group">
          <label for="homa-prechat-message">${escapeHtml(t('messageLabel'))} <span class="required">${t('nameRequired')}</span></label>
          <textarea id="homa-prechat-message" class="homa-chat-form-textarea" placeholder="${escapeHtml(t('messagePlaceholder'))}" required rows="2"></textarea>
        </div>`;
    }

    return html;
  }

  function createWidget() {
    // Get translations with config overrides
    const greetingTitle = config.greetingTitle || t('greetingTitle');
    const greetingMessage = config.greetingMessage || t('greetingMessage');
    const launcherText = config.launcherText || t('launcherText');

    // Default chat icon or custom SVG
    const defaultIcon = `<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
        <path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H6l-2 2V4h16v12z"/>
      </svg>`;
    const launcherIcon = config.launcherIcon || defaultIcon;

    // Create container
    const container = document.createElement('div');
    container.className = 'homa-chat-widget';
    container.id = 'homa-chat-widget';

    // Launcher button
    const launcher = document.createElement('button');
    launcher.className = 'homa-chat-launcher';
    launcher.setAttribute('aria-label', launcherText);
    launcher.innerHTML = `
      ${launcherIcon}
      <span class="homa-chat-launcher-badge">0</span>
    `;
    launcher.onclick = toggleWidget;

    // Chat window
    const chatWindow = document.createElement('div');
    chatWindow.className = 'homa-chat-window';
    chatWindow.innerHTML = `
      <div class="homa-chat-header">
        <div class="homa-chat-header-info">
          <div class="homa-chat-header-avatar">
            <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 3c1.66 0 3 1.34 3 3s-1.34 3-3 3-3-1.34-3-3 1.34-3 3-3zm0 14.2c-2.5 0-4.71-1.28-6-3.22.03-1.99 4-3.08 6-3.08 1.99 0 5.97 1.09 6 3.08-1.29 1.94-3.5 3.22-6 3.22z"/>
            </svg>
          </div>
          <div class="homa-chat-header-text">
            <h4>${escapeHtml(greetingTitle)}</h4>
            <p>${escapeHtml(t('replyTime'))}</p>
          </div>
        </div>
        <div class="homa-chat-header-buttons">
          <button class="homa-chat-minimize" aria-label="Minimize chat">
            <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
              <path d="M19 13H5v-2h14v2z"/>
            </svg>
          </button>
          <button class="homa-chat-close" aria-label="End chat" title="${escapeHtml(t('endChat') || 'End chat')}">
            <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
              <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
            </svg>
          </button>
        </div>
      </div>

      <!-- Pre-chat Form -->
      <div class="homa-chat-prechat-form">
        <div class="homa-chat-prechat-intro">
          <h5>${escapeHtml(greetingTitle)}</h5>
          <p>${escapeHtml(greetingMessage)}</p>
        </div>
        <form class="homa-chat-form" onsubmit="return false;">
          ${buildPreChatFormFields()}
          <button type="submit" class="homa-chat-form-submit">${escapeHtml(t('startChat'))}</button>
        </form>
      </div>

      <!-- Chat Area (hidden initially) -->
      <div class="homa-chat-messages" style="display: none;">
        <div class="homa-chat-greeting">
          <h5>${escapeHtml(greetingTitle)}</h5>
          <p>${escapeHtml(greetingMessage)}</p>
        </div>
        <div class="homa-chat-typing">
          <div class="homa-chat-typing-dots">
            <span></span><span></span><span></span>
          </div>
        </div>
      </div>
      <div class="homa-chat-input-container" style="display: none;">
        <textarea
          class="homa-chat-input"
          placeholder="${escapeHtml(t('typeMessage'))}"
          rows="1"
          aria-label="${escapeHtml(t('sendMessage'))}"
        ></textarea>
        <button class="homa-chat-send" aria-label="${escapeHtml(t('sendMessage'))}">
          <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
            <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/>
          </svg>
        </button>
      </div>
      <div class="homa-chat-powered ${config.hidePoweredBy ? 'hidden' : ''}">
        ${escapeHtml(t('poweredBy'))} <a href="https://github.com/evocert/homa" target="_blank">Homa</a>
      </div>
    `;

    container.appendChild(launcher);
    container.appendChild(chatWindow);
    document.body.appendChild(container);

    // Store element references
    elements = {
      container,
      launcher,
      chatWindow,
      badge: launcher.querySelector('.homa-chat-launcher-badge'),
      closeBtn: chatWindow.querySelector('.homa-chat-close'),
      messagesContainer: chatWindow.querySelector('.homa-chat-messages'),
      typingIndicator: chatWindow.querySelector('.homa-chat-typing'),
      input: chatWindow.querySelector('.homa-chat-input'),
      sendBtn: chatWindow.querySelector('.homa-chat-send'),
      inputContainer: chatWindow.querySelector('.homa-chat-input-container'),
      powered: chatWindow.querySelector('.homa-chat-powered'),
      minimizeBtn: chatWindow.querySelector('.homa-chat-minimize'),
      // Pre-chat form elements
      prechatForm: chatWindow.querySelector('.homa-chat-prechat-form'),
      prechatFormEl: chatWindow.querySelector('.homa-chat-form'),
      prechatName: chatWindow.querySelector('#homa-prechat-name'),
      prechatEmail: chatWindow.querySelector('#homa-prechat-email'),
      prechatPhone: chatWindow.querySelector('#homa-prechat-phone'),
      prechatCompany: chatWindow.querySelector('#homa-prechat-company'),
      prechatDepartment: chatWindow.querySelector('#homa-prechat-department'),
      prechatMessage: chatWindow.querySelector('#homa-prechat-message'),
      prechatSubmit: chatWindow.querySelector('.homa-chat-form-submit')
    };

    // Add event listeners
    elements.minimizeBtn.onclick = closeWidget;
    elements.closeBtn.onclick = handleCloseButton;
    elements.sendBtn.onclick = handleSend;
    elements.input.addEventListener('keydown', handleKeyDown);
    elements.input.addEventListener('input', handleInputChange);

    // Pre-chat form listeners
    elements.prechatFormEl.addEventListener('submit', handlePrechatSubmit);
    elements.prechatSubmit.onclick = handlePrechatSubmit;

    // Load departments for dropdown
    loadDepartments();

    // Pre-fill user info if provided
    if (config.userName && elements.prechatName) {
      elements.prechatName.value = config.userName;
    }
    if (config.userEmail && elements.prechatEmail) {
      elements.prechatEmail.value = config.userEmail;
    }

    // Store user attributes and conversation attributes in customAttributes
    if (config.userAttributes && typeof config.userAttributes === 'object') {
      customAttributes = { ...customAttributes, ...config.userAttributes };
    }
    if (config.conversationAttributes && typeof config.conversationAttributes === 'object') {
      customAttributes = { ...customAttributes, ...config.conversationAttributes };
    }
  }

  function handleKeyDown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  function handleInputChange() {
    const input = elements.input;
    // Auto-resize textarea
    input.style.height = 'auto';
    input.style.height = Math.min(input.scrollHeight, 120) + 'px';

    // Enable/disable send button
    elements.sendBtn.disabled = !input.value.trim();
  }

  // Load departments for the dropdown
  async function loadDepartments() {
    try {
      const response = await apiRequest('/api/system/departments');
      const departments = response.data || [];

      if (departments.length > 0 && elements.prechatDepartment) {
        // Show department selector with all options
        elements.prechatDepartment.innerHTML = '<option value="">Select a department...</option>';

        departments.forEach(dept => {
          const option = document.createElement('option');
          option.value = dept.id;
          option.textContent = dept.name;
          elements.prechatDepartment.appendChild(option);
        });

        // Pre-select the default department if configured
        if (config.defaultDepartmentId) {
          elements.prechatDepartment.value = String(config.defaultDepartmentId);
        }
      }
    } catch (error) {
      console.warn('HomaChat: Could not load departments', error);
      // Hide department field if we can't load departments
      const deptGroup = elements.prechatDepartment?.closest('.homa-chat-form-group');
      if (deptGroup) {
        deptGroup.style.display = 'none';
      }
    }
  }

  // Handle pre-chat form submission
  async function handlePrechatSubmit(e) {
    if (e) e.preventDefault();

    const name = elements.prechatName?.value?.trim() || '';
    const email = elements.prechatEmail?.value?.trim() || '';
    const departmentId = elements.prechatDepartment?.value || '';
    const message = elements.prechatMessage?.value?.trim() || '';

    // Collect custom field values
    const customFieldValues = {};
    const customFieldDefs = config.customFields || [];

    customFieldDefs.forEach(field => {
      const input = document.getElementById(`homa-prechat-${field.name}`);
      if (input) {
        let value = input.value?.trim() || '';
        // Type conversion
        if (value && (field.type === 'int' || field.data_type === 'int')) {
          value = parseInt(value) || 0;
        } else if (value && (field.type === 'float' || field.data_type === 'float')) {
          value = parseFloat(value) || 0;
        }
        if (value) {
          customFieldValues[field.name] = value;
        }
      }
    });

    // Validate required fields based on preChatFormFields
    const fields = config.preChatFormFields || ['name', 'email'];
    if (fields.includes('name') && !name && elements.prechatName) {
      elements.prechatName.focus();
      return;
    }
    if (!message && elements.prechatMessage) {
      elements.prechatMessage.focus();
      return;
    }

    // Validate required custom fields
    for (const field of customFieldDefs) {
      if (field.required && fields.includes(field.name)) {
        const value = customFieldValues[field.name];
        if (!value && value !== 0) {
          const input = document.getElementById(`homa-prechat-${field.name}`);
          if (input) {
            input.focus();
            return;
          }
        }
      }
    }

    // Disable submit button
    elements.prechatSubmit.disabled = true;
    elements.prechatSubmit.textContent = t('startingChat');

    try {
      // Set user info with additional fields
      user = {
        name: name,
        email: email || null,
        attributes: config.userAttributes || {}
      };
      safeLocalStorage('set', STORAGE_KEYS.USER, user);

      // Store department if selected
      if (departmentId) {
        customAttributes.department_id = parseInt(departmentId);
      }

      // Add custom field values to attributes
      Object.assign(customAttributes, customFieldValues);

      // Add conversation attributes from config
      if (config.conversationAttributes && typeof config.conversationAttributes === 'object') {
        Object.assign(customAttributes, config.conversationAttributes);
      }

      // Create conversation with the first message
      await createConversationWithMessage(message, departmentId ? parseInt(departmentId) : null);

      // Switch to chat view
      showChatView();

      emit('prechat:submitted', { name, email, departmentId, message, customFields: customFieldValues });
    } catch (error) {
      console.error('HomaChat: Failed to start chat', error);
      elements.prechatSubmit.disabled = false;
      elements.prechatSubmit.textContent = t('startChat');
      // Could show error message to user here
    }
  }

  // Create conversation with initial message
  async function createConversationWithMessage(initialMessage, departmentId) {
    const clientName = user?.name || 'Website Visitor';
    const clientEmail = user?.email || null;
    const clientPhone = user?.phone || null;

    const payload = {
      title: `Chat from ${clientName}`,
      status: 'new',
      priority: 'medium',
      client_name: clientName,
      client_email: clientEmail,
      client_phone: clientPhone,
      client_attributes: { ...customAttributes, ...user?.attributes },
      parameters: {
        source: 'widget',
        page_url: window.location.href,
        referrer: document.referrer,
        user_agent: navigator.userAgent
      }
    };

    // Add department if specified, fall back to defaultDepartmentId from config
    const deptId = departmentId || config.defaultDepartmentId;
    if (deptId) {
      payload.department_id = deptId;
    }

    try {
      const response = await apiRequest('/api/client/conversations', 'PUT', payload);
      conversation = {
        id: response.data.id,
        secret: response.data.secret
      };

      // Store conversation
      safeLocalStorage('set', STORAGE_KEYS.CONVERSATION, conversation);

      // Connect WebSocket
      connectWebSocket();

      emit('conversation:created', conversation);

      // Send the initial message
      if (initialMessage) {
        await sendMessage(initialMessage);
      }

      return conversation;
    } catch (error) {
      emit('error', { type: 'conversation_create', error });
      throw error;
    }
  }

  // Switch from pre-chat form to chat view
  function showChatView() {
    if (elements.prechatForm) {
      elements.prechatForm.style.display = 'none';
    }
    if (elements.messagesContainer) {
      elements.messagesContainer.style.display = 'flex';
    }
    if (elements.inputContainer) {
      elements.inputContainer.style.display = 'flex';
    }
    if (elements.input) {
      elements.input.focus();
    }
    // Scroll to bottom after switching views
    scrollToBottom();
  }

  // Switch from chat view to pre-chat form
  function showPrechatForm() {
    if (elements.prechatForm) {
      elements.prechatForm.style.display = 'block';
      // Reset form
      if (elements.prechatFormEl) {
        elements.prechatFormEl.reset();
        // Re-fill user info if provided in config
        if (config.userName && elements.prechatName) {
          elements.prechatName.value = config.userName;
        }
        if (config.userEmail && elements.prechatEmail) {
          elements.prechatEmail.value = config.userEmail;
        }
      }
      if (elements.prechatSubmit) {
        elements.prechatSubmit.disabled = false;
        elements.prechatSubmit.textContent = t('startChat');
      }
    }
    if (elements.messagesContainer) {
      elements.messagesContainer.style.display = 'none';
    }
    if (elements.inputContainer) {
      elements.inputContainer.style.display = 'none';
    }
  }

  // Handle close button click - ends chat if active, otherwise just closes
  function handleCloseButton() {
    if (conversation) {
      // Active conversation - confirm and end chat
      handleEndChat();
    } else {
      // No active conversation - just close widget
      closeWidget();
    }
  }

  async function handleSend() {
    const input = elements.input;
    const message = input.value.trim();

    if (!message) return;

    input.value = '';
    input.style.height = 'auto';
    elements.sendBtn.disabled = true;

    try {
      await sendMessage(message);
    } catch (error) {
      // Show error in UI
      console.error('Failed to send message:', error);
    }
  }

  async function handleEndChat() {
    // Confirm with user
    const confirmMsg = t('endChatConfirm') || 'Are you sure you want to end this chat?';
    if (!confirm(confirmMsg)) {
      return;
    }

    // Close the conversation on the backend
    if (conversation) {
      try {
        const endpoint = `/api/client/conversations/${conversation.id}/${conversation.secret}`;
        await apiRequest(endpoint, 'PATCH', { status: 'closed' });
      } catch (error) {
        console.error('HomaChat: Failed to close conversation', error);
        // Continue with local reset even if backend call fails
      }
    }

    // Disconnect WebSocket
    disconnectWebSocket();

    // Clear local state
    conversation = null;
    messages = [];
    safeLocalStorage('remove', STORAGE_KEYS.CONVERSATION);
    safeLocalStorage('remove', STORAGE_KEYS.MESSAGES);

    // Reset UI to pre-chat form
    showPrechatForm();

    // Clear messages from container
    if (elements.messagesContainer) {
      const greeting = elements.messagesContainer.querySelector('.homa-chat-greeting');
      if (greeting) greeting.style.display = 'block';
      const msgs = elements.messagesContainer.querySelectorAll('.homa-chat-message, .homa-chat-message-wrapper');
      msgs.forEach(el => el.remove());
    }

    emit('chat:ended');
  }

  function renderMessages() {
    const container = elements.messagesContainer;
    const greeting = container.querySelector('.homa-chat-greeting');
    const typing = container.querySelector('.homa-chat-typing');

    // Remove existing messages but keep greeting and typing indicator
    const existingMessages = container.querySelectorAll('.homa-chat-message, .homa-chat-message-wrapper');
    existingMessages.forEach(el => el.remove());

    // Hide greeting if there are messages
    if (messages.length > 0 && greeting) {
      greeting.style.display = 'none';
    }

    // Render messages
    messages.forEach(msg => {
      const msgEl = document.createElement('div');

      if (msg.is_client) {
        // Client messages: simple bubble, no avatar/name
        msgEl.className = 'homa-chat-message client';
        msgEl.innerHTML = escapeHtml(msg.body);
      } else {
        // Agent messages: show avatar and name
        msgEl.className = 'homa-chat-message-wrapper agent';

        const name = msg.user_name || 'Support Agent';
        const initials = name.split(' ').map(n => n[0]).join('').substring(0, 2).toUpperCase();
        const avatarUrl = msg.user_avatar ? (msg.user_avatar.startsWith('http') ? msg.user_avatar : config.baseUrl + '/media/' + msg.user_avatar) : null;

        msgEl.innerHTML = `
          <div class="homa-chat-message-header">
            <div class="homa-chat-avatar ${avatarUrl ? '' : 'initials'}" ${avatarUrl ? `style="background-image: url('${avatarUrl}')"` : ''}>
              ${avatarUrl ? '' : initials}
            </div>
            <span class="homa-chat-sender-name">${escapeHtml(name)}</span>
            <span class="homa-chat-message-time">${formatTime(msg.created_at)}</span>
          </div>
          <div class="homa-chat-message agent">
            ${escapeHtml(msg.body)}
          </div>
        `;
      }

      container.insertBefore(msgEl, typing);
    });

    // Scroll to bottom
    scrollToBottom();
  }

  function showTypingIndicator(show) {
    if (elements.typingIndicator) {
      elements.typingIndicator.classList.toggle('visible', show);
    }
  }

  function showNotification() {
    // Update badge
    if (elements.badge) {
      const count = parseInt(elements.badge.textContent) + 1;
      elements.badge.textContent = count;
      elements.badge.style.display = 'flex';
    }

    // Play sound if allowed
    // Could add notification sound here

    emit('notification', { unreadCount: parseInt(elements.badge?.textContent || '0') });
  }

  function clearNotifications() {
    if (elements.badge) {
      elements.badge.textContent = '0';
      elements.badge.style.display = 'none';
    }
  }

  // ==========================================
  // Widget Control Functions
  // ==========================================

  function openWidget() {
    if (isOpen) return;

    isOpen = true;
    elements.chatWindow.classList.add('open');
    clearNotifications();
    restoreTitle();

    // Focus on the appropriate element based on current view
    if (elements.prechatForm && elements.prechatForm.style.display !== 'none') {
      // Pre-chat form is visible, focus on name field
      if (elements.prechatName) {
        elements.prechatName.focus();
      }
    } else {
      // Chat view is visible, focus on message input
      if (elements.input) {
        elements.input.focus();
      }
      // Scroll to bottom when opening with existing messages
      scrollToBottom();
    }

    // Connect WebSocket if we have a conversation
    if (conversation && !websocket) {
      connectWebSocket();
    }

    emit('widget:opened');
  }

  function closeWidget() {
    if (!isOpen) return;

    isOpen = false;
    elements.chatWindow.classList.remove('open');
    emit('widget:closed');
  }

  function toggleWidget() {
    if (isOpen) {
      closeWidget();
    } else {
      openWidget();
    }
  }

  // ==========================================
  // Public API
  // ==========================================

  function init(options) {
    console.log('HomaChat: init() called with', options);

    if (isInitialized) {
      console.warn('HomaChat: Already initialized');
      return;
    }

    // Merge config
    config = { ...DEFAULT_CONFIG, ...options };
    console.log('HomaChat: config merged', config);

    // Validate required options
    if (!config.baseUrl) {
      console.error('HomaChat: baseUrl is required');
      return;
    }

    // Restore session
    conversation = safeLocalStorage('get', STORAGE_KEYS.CONVERSATION);
    user = safeLocalStorage('get', STORAGE_KEYS.USER);
    messages = safeLocalStorage('get', STORAGE_KEYS.MESSAGES) || [];

    // Create UI
    console.log('HomaChat: Creating UI...');
    createStyles();
    console.log('HomaChat: Styles created');
    createWidget();
    console.log('HomaChat: Widget created');

    // Check if there's an existing conversation
    if (conversation) {
      // Show chat view instead of pre-chat form
      showChatView();
      // Render cached messages first for quick display
      if (messages.length > 0) {
        renderMessages();
      }
      // Always load fresh messages from API to get updated user info
      loadMessages();
      // Connect WebSocket
      connectWebSocket();
    }
    // Otherwise, pre-chat form is already visible by default

    isInitialized = true;
    emit('ready');

    // Inject custom CSS if provided
    if (config.customCSS) {
      injectCustomCSS(config.customCSS);
    }

    // Auto-open functionality
    if (config.autoOpen > 0) {
      handleAutoOpen();
    }

    console.log('HomaChat: Initialized');
  }

  // Inject custom CSS
  function injectCustomCSS(css) {
    try {
      const styleEl = document.createElement('style');
      styleEl.id = 'homa-chat-custom-css';
      styleEl.textContent = css;
      document.head.appendChild(styleEl);
    } catch (e) {
      console.warn('HomaChat: Failed to inject custom CSS', e);
    }
  }

  // Handle auto-open
  function handleAutoOpen() {
    const sessionKey = 'homa_chat_auto_opened';

    // Check if already auto-opened this session (if autoOpenOnce is true)
    if (config.autoOpenOnce) {
      try {
        if (sessionStorage.getItem(sessionKey)) {
          return;
        }
      } catch (e) {
        // sessionStorage not available, continue
      }
    }

    setTimeout(() => {
      if (!isOpen) {
        openWidget();
        // Mark as auto-opened for this session
        if (config.autoOpenOnce) {
          try {
            sessionStorage.setItem(sessionKey, 'true');
          } catch (e) {
            // sessionStorage not available
          }
        }
      }
    }, config.autoOpen);
  }

  function setUser(userData) {
    user = {
      name: userData.name,
      email: userData.email,
      phone: userData.phone,
      identifier: userData.identifier,
      attributes: userData.attributes || {}
    };
    safeLocalStorage('set', STORAGE_KEYS.USER, user);
    emit('user:set', user);
  }

  function setCustomAttributes(attrs) {
    customAttributes = { ...customAttributes, ...attrs };
    emit('attributes:set', customAttributes);
  }

  function reset() {
    // Clear everything
    conversation = null;
    messages = [];
    user = null;
    customAttributes = {};

    safeLocalStorage('remove', STORAGE_KEYS.CONVERSATION);
    safeLocalStorage('remove', STORAGE_KEYS.USER);
    safeLocalStorage('remove', STORAGE_KEYS.MESSAGES);

    disconnectWebSocket();

    // Reset UI
    if (elements.messagesContainer) {
      const greeting = elements.messagesContainer.querySelector('.homa-chat-greeting');
      if (greeting) greeting.style.display = 'block';
      const msgs = elements.messagesContainer.querySelectorAll('.homa-chat-message');
      msgs.forEach(el => el.remove());
    }

    // Show pre-chat form again
    showPrechatForm();

    clearNotifications();
    closeWidget();

    emit('reset');
  }

  function on(eventName, callback) {
    if (!eventCallbacks[eventName]) {
      eventCallbacks[eventName] = [];
    }
    eventCallbacks[eventName].push(callback);
  }

  function off(eventName, callback) {
    if (eventCallbacks[eventName]) {
      eventCallbacks[eventName] = eventCallbacks[eventName].filter(cb => cb !== callback);
    }
  }

  // Process queued commands
  function processQueue() {
    const queue = window.homaChat?.q || [];
    console.log('HomaChat: processQueue() called, queue length:', queue.length);
    queue.forEach((queuedArgs, index) => {
      // Convert Arguments object to array
      const args = Array.prototype.slice.call(queuedArgs);
      const method = args[0];
      const params = args.slice(1);
      console.log('HomaChat: Processing queue item', index, '- method:', method);

      switch (method) {
        case 'init':
          init(params[0]);
          break;
        case 'setUser':
          setUser(params[0]);
          break;
        case 'setCustomAttributes':
          setCustomAttributes(params[0]);
          break;
        case 'open':
          openWidget();
          break;
        case 'close':
          closeWidget();
          break;
        case 'toggle':
          toggleWidget();
          break;
        case 'reset':
          reset();
          break;
        case 'on':
          on(params[0], params[1]);
          break;
        case 'off':
          off(params[0], params[1]);
          break;
        default:
          console.warn('HomaChat: Unknown method', method);
      }
    });
  }

  // Expose public API
  const publicAPI = function(method, ...args) {
    switch (method) {
      case 'init':
        init(args[0]);
        break;
      case 'setUser':
        setUser(args[0]);
        break;
      case 'setCustomAttributes':
        setCustomAttributes(args[0]);
        break;
      case 'open':
        openWidget();
        break;
      case 'close':
        closeWidget();
        break;
      case 'toggle':
        toggleWidget();
        break;
      case 'reset':
        reset();
        break;
      case 'on':
        on(args[0], args[1]);
        break;
      case 'off':
        off(args[0], args[1]);
        break;
      case 'sendMessage':
        return sendMessage(args[0]);
      default:
        console.warn('HomaChat: Unknown method', method);
    }
  };

  // Save the queue before replacing the function
  const savedQueue = window.homaChat?.q || [];
  console.log('HomaChat: Saved queue length:', savedQueue.length);

  // Replace queue function with actual API
  window.homaChat = publicAPI;

  // Process any queued commands when DOM is ready
  function runInit() {
    console.log('HomaChat: runInit() called, DOM ready');
    // Process the saved queue
    savedQueue.forEach((queuedArgs, index) => {
      const args = Array.prototype.slice.call(queuedArgs);
      const method = args[0];
      const params = args.slice(1);
      console.log('HomaChat: Processing queue item', index, '- method:', method);

      switch (method) {
        case 'init':
          init(params[0]);
          break;
        case 'setUser':
          setUser(params[0]);
          break;
        case 'setCustomAttributes':
          setCustomAttributes(params[0]);
          break;
        case 'open':
          openWidget();
          break;
        case 'close':
          closeWidget();
          break;
        case 'toggle':
          toggleWidget();
          break;
        case 'reset':
          reset();
          break;
        case 'on':
          on(params[0], params[1]);
          break;
        case 'off':
          off(params[0], params[1]);
          break;
        default:
          console.warn('HomaChat: Unknown method', method);
      }
    });
  }

  console.log('HomaChat: SDK loaded, readyState:', document.readyState);

  if (document.readyState === 'loading') {
    console.log('HomaChat: Waiting for DOMContentLoaded');
    document.addEventListener('DOMContentLoaded', runInit);
  } else {
    // DOM is already ready
    console.log('HomaChat: DOM already ready, running init');
    runInit();
  }

  // Also expose as HomaChat for alternative access
  window.HomaChat = {
    init,
    setUser,
    setCustomAttributes,
    open: openWidget,
    close: closeWidget,
    toggle: toggleWidget,
    reset,
    on,
    off,
    sendMessage
  };

})(window, document);
