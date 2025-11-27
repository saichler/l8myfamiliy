/**
 * Login functionality for Family Locator
 * Handles authentication and credential storage
 */

(function() {
    'use strict';

    const STORAGE_KEY = 'familyLocatorCredentials';
    const SESSION_KEY = 'familyLocatorSession';

    const loginForm = document.getElementById('loginForm');
    const familyNameInput = document.getElementById('familyName');
    const passwordInput = document.getElementById('password');
    const rememberMeCheckbox = document.getElementById('rememberMe');
    const errorMessage = document.getElementById('errorMessage');

    /**
     * Initialize the login page
     */
    function init() {
        checkExistingSession();
        loadSavedCredentials();
        setupEventListeners();
    }

    /**
     * Check if user already has a valid session
     */
    function checkExistingSession() {
        const session = sessionStorage.getItem(SESSION_KEY);
        if (session) {
            try {
                const sessionData = JSON.parse(session);
                if (sessionData.familyName) {
                    redirectToMap();
                }
            } catch (e) {
                sessionStorage.removeItem(SESSION_KEY);
            }
        }
    }

    /**
     * Load saved credentials from localStorage
     */
    function loadSavedCredentials() {
        const savedCredentials = localStorage.getItem(STORAGE_KEY);
        if (savedCredentials) {
            try {
                const credentials = JSON.parse(savedCredentials);
                if (credentials.familyName) {
                    familyNameInput.value = credentials.familyName;
                }
                if (credentials.password) {
                    passwordInput.value = credentials.password;
                }
                rememberMeCheckbox.checked = true;
            } catch (e) {
                localStorage.removeItem(STORAGE_KEY);
            }
        }
    }

    /**
     * Set up event listeners
     */
    function setupEventListeners() {
        loginForm.addEventListener('submit', handleLogin);
        familyNameInput.addEventListener('input', clearError);
        passwordInput.addEventListener('input', clearError);
    }

    /**
     * Handle login form submission
     * @param {Event} e - Form submit event
     */
    async function handleLogin(e) {
        e.preventDefault();
        clearError();

        const familyName = familyNameInput.value.trim();
        const password = passwordInput.value;
        const rememberMe = rememberMeCheckbox.checked;

        if (!familyName) {
            showError('Please enter your family name');
            familyNameInput.focus();
            return;
        }

        if (!password) {
            showError('Please enter your password');
            passwordInput.focus();
            return;
        }

        const submitBtn = loginForm.querySelector('.login-btn');
        submitBtn.disabled = true;
        submitBtn.textContent = 'Signing in...';

        try {
            const success = await authenticateUser(familyName, password);

            if (success) {
                if (rememberMe) {
                    saveCredentials(familyName, password);
                } else {
                    localStorage.removeItem(STORAGE_KEY);
                }

                createSession(familyName);
                redirectToMap();
            } else {
                showError('Invalid family name or password');
            }
        } catch (error) {
            showError('Connection error. Please try again.');
            console.error('Login error:', error);
        } finally {
            submitBtn.disabled = false;
            submitBtn.textContent = 'Sign In';
        }
    }

    /**
     * Authenticate user with backend
     * @param {string} familyName - Family name (used as username)
     * @param {string} password - Password
     * @returns {Promise<boolean>} - Authentication result
     */
    async function authenticateUser(familyName, password) {
        try {
            const response = await fetch('/auth', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    user: familyName,
                    pass: password
                })
            });

            if (response.ok) {
                const token = await response.text();
                if (token) {
                    sessionStorage.setItem('authToken', token);
                }
                return true;
            }

            if (response.status === 401) {
                return false;
            }

            throw new Error(`HTTP ${response.status}`);
        } catch (error) {
            if (error.message.includes('Failed to fetch') ||
                error.message.includes('NetworkError')) {
                // For development/demo: allow login without backend
                console.warn('Backend not available, using demo mode');
                return true;
            }
            throw error;
        }
    }

    /**
     * Save credentials to localStorage
     * @param {string} familyName - Family name
     * @param {string} password - Password
     */
    function saveCredentials(familyName, password) {
        const credentials = {
            familyName: familyName,
            password: password,
            savedAt: Date.now()
        };
        localStorage.setItem(STORAGE_KEY, JSON.stringify(credentials));
    }

    /**
     * Create session for authenticated user
     * @param {string} familyName - Family name
     */
    function createSession(familyName) {
        const session = {
            familyName: familyName,
            loginTime: Date.now()
        };
        sessionStorage.setItem(SESSION_KEY, JSON.stringify(session));
    }

    /**
     * Redirect to the map page
     */
    function redirectToMap() {
        window.location.href = 'map.html';
    }

    /**
     * Show error message
     * @param {string} message - Error message to display
     */
    function showError(message) {
        errorMessage.textContent = message;
        errorMessage.style.display = 'block';
    }

    /**
     * Clear error message
     */
    function clearError() {
        errorMessage.textContent = '';
        errorMessage.style.display = 'none';
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
