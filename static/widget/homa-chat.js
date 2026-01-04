/**
 * Homa Chat Widget SDK
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
 *       baseUrl: 'https://api.example.com'
 *     });
 *   </script>
 */

(function(window, document) {
  'use strict';

  // Default configuration
  const DEFAULT_CONFIG = {
    baseUrl: '',
    websiteToken: '',
    position: 'right', // 'left' or 'right'
    launcherText: 'Chat with us',
    brandColor: '#3B82F6', // Primary blue
    textColor: '#FFFFFF',
    greetingMessage: 'Hi! How can we help you today?',
    greetingTitle: 'Welcome',
    showAvatar: true,
    hideOnMobile: false,
    locale: 'en',
    darkMode: false,
    zIndex: 999999
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

  // Storage keys
  const STORAGE_KEYS = {
    CONVERSATION: 'homa_chat_conversation',
    USER: 'homa_chat_user',
    MESSAGES: 'homa_chat_messages'
  };

  // ==========================================
  // Utility Functions
  // ==========================================

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
    return d.toLocaleTimeString(config.locale, { hour: '2-digit', minute: '2-digit' });
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
      credentials: 'include'
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
      messages = response.data?.messages || [];
      safeLocalStorage('set', STORAGE_KEYS.MESSAGES, messages);
      renderMessages();
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
    // Handle different message types
    if (data.type === 'message.created' || data.event === 'message_created') {
      const message = data.message || data.data;
      if (message && !message.is_client) {
        messages.push({
          id: message.id,
          body: message.body,
          is_client: false,
          user_name: message.user?.name || 'Support Agent',
          created_at: message.created_at
        });
        renderMessages();
        safeLocalStorage('set', STORAGE_KEYS.MESSAGES, messages);
        emit('message:received', message);

        // Show notification if widget is closed
        if (!isOpen) {
          showNotification();
        }
      }
    } else if (data.type === 'typing') {
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
      }

      .homa-chat-launcher {
        width: 60px;
        height: 60px;
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
        width: 28px;
        height: 28px;
        fill: ${config.textColor};
      }

      .homa-chat-launcher-badge {
        position: absolute;
        top: -5px;
        right: -5px;
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
        bottom: 80px;
        ${config.position}: 0;
        width: 380px;
        height: 550px;
        max-height: calc(100vh - 120px);
        background-color: ${config.darkMode ? '#1F2937' : '#FFFFFF'};
        border-radius: 16px;
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
        background-color: ${config.brandColor};
        color: ${config.textColor};
        padding: 16px 20px;
        display: flex;
        align-items: center;
        justify-content: space-between;
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

      .homa-chat-close {
        background: none;
        border: none;
        cursor: pointer;
        padding: 8px;
        border-radius: 8px;
        transition: background-color 0.2s;
      }

      .homa-chat-close:hover {
        background-color: rgba(255, 255, 255, 0.1);
      }

      .homa-chat-close svg {
        width: 20px;
        height: 20px;
        fill: ${config.textColor};
      }

      .homa-chat-messages {
        flex: 1;
        overflow-y: auto;
        padding: 20px;
        display: flex;
        flex-direction: column;
        gap: 12px;
        background-color: ${config.darkMode ? '#111827' : '#F9FAFB'};
      }

      .homa-chat-greeting {
        text-align: center;
        padding: 20px;
        color: ${config.darkMode ? '#9CA3AF' : '#6B7280'};
      }

      .homa-chat-greeting h5 {
        font-size: 18px;
        font-weight: 600;
        margin-bottom: 8px;
        color: ${config.darkMode ? '#F9FAFB' : '#111827'};
      }

      .homa-chat-message {
        max-width: 80%;
        padding: 12px 16px;
        border-radius: 16px;
        line-height: 1.5;
        word-wrap: break-word;
      }

      .homa-chat-message.client {
        align-self: flex-end;
        background-color: ${config.brandColor};
        color: ${config.textColor};
        border-bottom-right-radius: 4px;
      }

      .homa-chat-message.agent {
        align-self: flex-start;
        background-color: ${config.darkMode ? '#374151' : '#FFFFFF'};
        color: ${config.darkMode ? '#F9FAFB' : '#111827'};
        border-bottom-left-radius: 4px;
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
      }

      .homa-chat-message-meta {
        font-size: 11px;
        opacity: 0.7;
        margin-top: 4px;
      }

      .homa-chat-typing {
        display: none;
        align-self: flex-start;
        padding: 12px 16px;
        background-color: ${config.darkMode ? '#374151' : '#FFFFFF'};
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
        background-color: ${config.darkMode ? '#9CA3AF' : '#6B7280'};
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
        background-color: ${config.darkMode ? '#1F2937' : '#FFFFFF'};
        border-top: 1px solid ${config.darkMode ? '#374151' : '#E5E7EB'};
        display: flex;
        gap: 12px;
        align-items: flex-end;
      }

      .homa-chat-input {
        flex: 1;
        padding: 12px 16px;
        border: 1px solid ${config.darkMode ? '#374151' : '#E5E7EB'};
        border-radius: 24px;
        font-size: 14px;
        resize: none;
        outline: none;
        max-height: 120px;
        background-color: ${config.darkMode ? '#374151' : '#F9FAFB'};
        color: ${config.darkMode ? '#F9FAFB' : '#111827'};
        transition: border-color 0.2s;
      }

      .homa-chat-input:focus {
        border-color: ${config.brandColor};
      }

      .homa-chat-input::placeholder {
        color: ${config.darkMode ? '#9CA3AF' : '#9CA3AF'};
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
        background-color: ${config.darkMode ? '#4B5563' : '#D1D5DB'};
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
        color: ${config.darkMode ? '#6B7280' : '#9CA3AF'};
        background-color: ${config.darkMode ? '#1F2937' : '#FFFFFF'};
      }

      .homa-chat-powered a {
        color: ${config.brandColor};
        text-decoration: none;
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

  function createWidget() {
    // Create container
    const container = document.createElement('div');
    container.className = 'homa-chat-widget';
    container.id = 'homa-chat-widget';

    // Launcher button
    const launcher = document.createElement('button');
    launcher.className = 'homa-chat-launcher';
    launcher.setAttribute('aria-label', config.launcherText);
    launcher.innerHTML = `
      <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
        <path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H6l-2 2V4h16v12z"/>
      </svg>
      <span class="homa-chat-launcher-badge">0</span>
    `;
    launcher.onclick = toggleWidget;

    // Chat window
    const window = document.createElement('div');
    window.className = 'homa-chat-window';
    window.innerHTML = `
      <div class="homa-chat-header">
        <div class="homa-chat-header-info">
          <div class="homa-chat-header-avatar">
            <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 3c1.66 0 3 1.34 3 3s-1.34 3-3 3-3-1.34-3-3 1.34-3 3-3zm0 14.2c-2.5 0-4.71-1.28-6-3.22.03-1.99 4-3.08 6-3.08 1.99 0 5.97 1.09 6 3.08-1.29 1.94-3.5 3.22-6 3.22z"/>
            </svg>
          </div>
          <div class="homa-chat-header-text">
            <h4>${config.greetingTitle}</h4>
            <p>We typically reply within a few minutes</p>
          </div>
        </div>
        <button class="homa-chat-close" aria-label="Close chat">
          <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
            <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
          </svg>
        </button>
      </div>
      <div class="homa-chat-messages">
        <div class="homa-chat-greeting">
          <h5>${config.greetingTitle}</h5>
          <p>${config.greetingMessage}</p>
        </div>
        <div class="homa-chat-typing">
          <div class="homa-chat-typing-dots">
            <span></span><span></span><span></span>
          </div>
        </div>
      </div>
      <div class="homa-chat-input-container">
        <textarea
          class="homa-chat-input"
          placeholder="Type your message..."
          rows="1"
          aria-label="Message input"
        ></textarea>
        <button class="homa-chat-send" aria-label="Send message">
          <svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
            <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/>
          </svg>
        </button>
      </div>
      <div class="homa-chat-powered">
        Powered by <a href="#" target="_blank">Homa</a>
      </div>
    `;

    container.appendChild(launcher);
    container.appendChild(window);
    document.body.appendChild(container);

    // Store element references
    elements = {
      container,
      launcher,
      window,
      badge: launcher.querySelector('.homa-chat-launcher-badge'),
      closeBtn: window.querySelector('.homa-chat-close'),
      messagesContainer: window.querySelector('.homa-chat-messages'),
      typingIndicator: window.querySelector('.homa-chat-typing'),
      input: window.querySelector('.homa-chat-input'),
      sendBtn: window.querySelector('.homa-chat-send')
    };

    // Add event listeners
    elements.closeBtn.onclick = closeWidget;
    elements.sendBtn.onclick = handleSend;
    elements.input.addEventListener('keydown', handleKeyDown);
    elements.input.addEventListener('input', handleInputChange);
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

  function renderMessages() {
    const container = elements.messagesContainer;
    const greeting = container.querySelector('.homa-chat-greeting');
    const typing = container.querySelector('.homa-chat-typing');

    // Remove existing messages but keep greeting and typing indicator
    const existingMessages = container.querySelectorAll('.homa-chat-message');
    existingMessages.forEach(el => el.remove());

    // Hide greeting if there are messages
    if (messages.length > 0 && greeting) {
      greeting.style.display = 'none';
    }

    // Render messages
    messages.forEach(msg => {
      const msgEl = document.createElement('div');
      msgEl.className = `homa-chat-message ${msg.is_client ? 'client' : 'agent'}`;
      msgEl.innerHTML = `
        ${escapeHtml(msg.body)}
        <div class="homa-chat-message-meta">
          ${msg.user_name ? msg.user_name + ' · ' : ''}${formatTime(msg.created_at)}
        </div>
      `;
      container.insertBefore(msgEl, typing);
    });

    // Scroll to bottom
    container.scrollTop = container.scrollHeight;
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
    elements.window.classList.add('open');
    elements.input.focus();
    clearNotifications();

    // Connect WebSocket if we have a conversation
    if (conversation && !websocket) {
      connectWebSocket();
    }

    emit('widget:opened');
  }

  function closeWidget() {
    if (!isOpen) return;

    isOpen = false;
    elements.window.classList.remove('open');
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
    if (isInitialized) {
      console.warn('HomaChat: Already initialized');
      return;
    }

    // Merge config
    config = { ...DEFAULT_CONFIG, ...options };

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
    createStyles();
    createWidget();

    // Render existing messages
    if (messages.length > 0) {
      renderMessages();
    }

    // Connect WebSocket if conversation exists
    if (conversation) {
      connectWebSocket();
    }

    isInitialized = true;
    emit('ready');

    console.log('HomaChat: Initialized');
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
    queue.forEach(args => {
      const method = args[0];
      const params = args.slice(1);

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

  // Replace queue function with actual API
  window.homaChat = publicAPI;

  // Process any queued commands
  processQueue();

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
