/**
 * Map functionality for Family Locator
 * Displays family member devices on a Leaflet map
 */

(function() {
    'use strict';

    const SESSION_KEY = 'familyLocatorSession';
    const REFRESH_INTERVAL = 30000; // 30 seconds

    let map = null;
    let markers = {};
    let devices = [];
    let familyName = '';
    let refreshTimer = null;

    // Color palette for members
    const MEMBER_COLORS = [
        '#e91e63', '#9c27b0', '#673ab7', '#3f51b5', '#2196f3',
        '#00bcd4', '#009688', '#4caf50', '#ff9800', '#ff5722'
    ];

    // Map member IDs to colors for consistency
    const memberColorMap = {};
    let colorIndex = 0;

    /**
     * Initialize the map page
     */
    function init() {
        if (!checkSession()) {
            return;
        }

        initializeUI();
        initializeMap();
        loadDevices();
        setupEventListeners();
        startRefreshTimer();
    }

    /**
     * Check if user has valid session
     * @returns {boolean} - Whether session is valid
     */
    function checkSession() {
        const session = sessionStorage.getItem(SESSION_KEY);
        if (!session) {
            redirectToLogin();
            return false;
        }

        try {
            const sessionData = JSON.parse(session);
            if (!sessionData.familyName) {
                redirectToLogin();
                return false;
            }
            familyName = sessionData.familyName;
            return true;
        } catch (e) {
            redirectToLogin();
            return false;
        }
    }

    /**
     * Redirect to login page
     */
    function redirectToLogin() {
        window.location.href = './';
    }

    /**
     * Initialize UI elements
     */
    function initializeUI() {
        const title = `The ${familyName} Family`;
        document.getElementById('familyTitle').textContent = title;
        document.getElementById('pageTitle').textContent = title;
    }

    /**
     * Initialize the Leaflet map
     */
    function initializeMap() {
        // Create map centered on a default location
        map = L.map('map', {
            zoomControl: true,
            attributionControl: true
        }).setView([37.7749, -122.4194], 12); // Default: San Francisco

        // Add OpenStreetMap tiles
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
            maxZoom: 19
        }).addTo(map);
    }

    /**
     * Load devices from API
     */
    async function loadDevices() {
        try {
            const response = await fetchDevices();
            devices = response;
            renderDevices();
            renderMemberList();
            fitMapToDevices();
        } catch (error) {
            console.error('Error loading devices:', error);
            // Show demo data if API is not available
            loadDemoDevices();
        }
    }

    /**
     * Fetch devices from backend API
     * @returns {Promise<Array>} - Array of MFDevice objects
     */
    async function fetchDevices() {
        const token = sessionStorage.getItem('authToken');
        const headers = {
            'Content-Type': 'application/json'
        };

        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        const query = `select * from Device where familyId=${familyName}`;
        const bodyParam = encodeURIComponent(JSON.stringify({ text: query }));
        const url = `/probler/53/Family?body=${bodyParam}`;

        const response = await fetch(url, {
            method: 'GET',
            headers: headers
        });

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }

        const data = await response.json();
        // Response format: {"list":[...], "metadata":{...}}
        const list = data.list || [];
        // Map API response to expected device format
        return list.map(device => ({
            id: device.id,
            name: device.name || device.id.substring(0, 8),
            memberId: device.memberId || device.familyId,
            memberName: device.memberName || device.name || 'Device',
            activity: device.activity || 'stationary',
            latitude: device.latitude,
            longitude: device.longitude
        }));
    }

    /**
     * Load demo devices for development/testing
     */
    function loadDemoDevices() {
        devices = [
            {
                id: 'device-1',
                name: 'iPhone 15 Pro',
                memberId: 'member-1',
                memberName: 'Dad',
                activity: 'driving',
                longitude: -122.4194,
                latitude: 37.7749
            },
            {
                id: 'device-2',
                name: 'Galaxy S24',
                memberId: 'member-2',
                memberName: 'Mom',
                activity: 'stationary',
                longitude: -122.4094,
                latitude: 37.7849
            },
            {
                id: 'device-3',
                name: 'iPhone 14',
                memberId: 'member-3',
                memberName: 'Emma',
                activity: 'walking',
                longitude: -122.4294,
                latitude: 37.7649
            },
            {
                id: 'device-4',
                name: 'Pixel 8',
                memberId: 'member-4',
                memberName: 'Jake',
                activity: 'running',
                longitude: -122.3994,
                latitude: 37.7949
            },
            {
                id: 'device-5',
                name: 'iPad Pro',
                memberId: 'member-1',
                memberName: 'Dad',
                activity: 'stationary',
                longitude: -122.4150,
                latitude: 37.7700
            }
        ];

        renderDevices();
        renderMemberList();
        fitMapToDevices();
    }

    /**
     * Get color for a member
     * @param {string} memberId - Member ID
     * @returns {string} - Color hex code
     */
    function getMemberColor(memberId) {
        if (!memberColorMap[memberId]) {
            memberColorMap[memberId] = MEMBER_COLORS[colorIndex % MEMBER_COLORS.length];
            colorIndex++;
        }
        return memberColorMap[memberId];
    }

    /**
     * Get initials from name
     * @param {string} name - Full name
     * @returns {string} - Initials
     */
    function getInitials(name) {
        if (!name) return '?';
        const parts = name.trim().split(/\s+/);
        if (parts.length === 1) {
            return parts[0].charAt(0).toUpperCase();
        }
        return (parts[0].charAt(0) + parts[parts.length - 1].charAt(0)).toUpperCase();
    }

    /**
     * Create custom marker icon
     * @param {Object} device - Device object
     * @returns {L.DivIcon} - Leaflet div icon
     */
    function createMarkerIcon(device) {
        const color = getMemberColor(device.memberId);
        const initials = getInitials(device.memberName);

        const html = `
            <div class="marker-pin" style="background-color: ${color};">
                ${initials}
            </div>
        `;

        return L.divIcon({
            html: html,
            className: 'custom-marker',
            iconSize: [36, 46],
            iconAnchor: [18, 46],
            popupAnchor: [0, -46]
        });
    }

    /**
     * Render devices on the map
     */
    function renderDevices() {
        // Clear existing markers
        Object.values(markers).forEach(marker => map.removeLayer(marker));
        markers = {};

        // Add markers for each device
        devices.forEach(device => {
            if (device.latitude && device.longitude) {
                const marker = L.marker([device.latitude, device.longitude], {
                    icon: createMarkerIcon(device)
                });

                const popupContent = `
                    <strong>${device.memberName}</strong>
                    <div>${device.name}</div>
                    <div style="color: #666; font-size: 0.85em;">${formatActivity(device.activity)}</div>
                `;

                marker.bindPopup(popupContent);
                marker.on('click', () => showDevicePopup(device));
                marker.addTo(map);
                markers[device.id] = marker;
            }
        });
    }

    /**
     * Render member list panel
     */
    function renderMemberList() {
        const memberList = document.getElementById('memberList');
        memberList.innerHTML = '';

        // Group devices by member
        const memberDevices = {};
        devices.forEach(device => {
            if (!memberDevices[device.memberId]) {
                memberDevices[device.memberId] = {
                    name: device.memberName,
                    devices: []
                };
            }
            memberDevices[device.memberId].devices.push(device);
        });

        // Create cards for each device
        devices.forEach(device => {
            const card = createMemberCard(device);
            memberList.appendChild(card);
        });
    }

    /**
     * Create a member card element
     * @param {Object} device - Device object
     * @returns {HTMLElement} - Card element
     */
    function createMemberCard(device) {
        const card = document.createElement('div');
        card.className = 'member-card';
        card.dataset.deviceId = device.id;

        const color = getMemberColor(device.memberId);
        const initials = getInitials(device.memberName);
        const activityClass = getActivityClass(device.activity);

        card.innerHTML = `
            <div class="member-avatar" style="background-color: ${color};">
                ${initials}
            </div>
            <div class="member-info">
                <div class="member-name">${device.memberName}</div>
                <div class="member-device">${device.name}</div>
            </div>
            <div class="member-activity ${activityClass}">
                ${formatActivity(device.activity)}
            </div>
        `;

        card.addEventListener('click', () => {
            centerOnDevice(device);
            showDevicePopup(device);
        });

        return card;
    }

    /**
     * Get activity CSS class
     * @param {string} activity - Activity type
     * @returns {string} - CSS class name
     */
    function getActivityClass(activity) {
        if (!activity) return '';
        const lower = activity.toLowerCase();
        if (lower.includes('driv')) return 'driving';
        if (lower.includes('walk')) return 'walking';
        if (lower.includes('run')) return 'running';
        return '';
    }

    /**
     * Format activity for display
     * @param {string} activity - Activity type
     * @returns {string} - Formatted activity
     */
    function formatActivity(activity) {
        if (!activity) return 'Unknown';
        return activity.charAt(0).toUpperCase() + activity.slice(1).toLowerCase();
    }

    /**
     * Center map on a device
     * @param {Object} device - Device object
     */
    function centerOnDevice(device) {
        if (device.latitude && device.longitude) {
            map.setView([device.latitude, device.longitude], 16);

            // Open the marker popup
            const marker = markers[device.id];
            if (marker) {
                marker.openPopup();
            }
        }
    }

    /**
     * Show device popup modal
     * @param {Object} device - Device object
     */
    function showDevicePopup(device) {
        const popup = document.getElementById('devicePopup');
        const color = getMemberColor(device.memberId);
        const initials = getInitials(device.memberName);

        document.getElementById('popupAvatar').innerHTML = initials;
        document.getElementById('popupAvatar').style.backgroundColor = color;
        document.getElementById('popupMemberName').textContent = device.memberName;
        document.getElementById('popupDeviceName').textContent = device.name;
        document.getElementById('popupActivity').textContent = formatActivity(device.activity);
        document.getElementById('popupLocation').textContent =
            `${device.latitude.toFixed(4)}, ${device.longitude.toFixed(4)}`;

        // Store device reference for center button
        popup.dataset.deviceId = device.id;
        popup.classList.remove('hidden');
    }

    /**
     * Hide device popup modal
     */
    function hideDevicePopup() {
        document.getElementById('devicePopup').classList.add('hidden');
    }

    /**
     * Fit map to show all devices
     */
    function fitMapToDevices() {
        if (devices.length === 0) return;

        const validDevices = devices.filter(d => d.latitude && d.longitude);
        if (validDevices.length === 0) return;

        if (validDevices.length === 1) {
            map.setView([validDevices[0].latitude, validDevices[0].longitude], 14);
            return;
        }

        const bounds = L.latLngBounds(
            validDevices.map(d => [d.latitude, d.longitude])
        );
        map.fitBounds(bounds, { padding: [50, 50] });
    }

    /**
     * Set up event listeners
     */
    function setupEventListeners() {
        // Logout button
        document.getElementById('logoutBtn').addEventListener('click', handleLogout);

        // Toggle member list
        document.getElementById('toggleMemberList').addEventListener('click', toggleMemberList);

        // Popup close button
        document.getElementById('closePopup').addEventListener('click', hideDevicePopup);

        // Center on device button in popup
        document.getElementById('centerOnDevice').addEventListener('click', () => {
            const deviceId = document.getElementById('devicePopup').dataset.deviceId;
            const device = devices.find(d => d.id === deviceId);
            if (device) {
                hideDevicePopup();
                centerOnDevice(device);
            }
        });

        // Close popup when clicking outside
        document.getElementById('devicePopup').addEventListener('click', (e) => {
            if (e.target.id === 'devicePopup') {
                hideDevicePopup();
            }
        });
    }

    /**
     * Handle logout
     */
    function handleLogout() {
        sessionStorage.removeItem(SESSION_KEY);
        sessionStorage.removeItem('authToken');
        stopRefreshTimer();
        redirectToLogin();
    }

    /**
     * Toggle member list visibility
     */
    function toggleMemberList() {
        const container = document.querySelector('.member-list-container');
        container.classList.toggle('collapsed');
    }

    /**
     * Start auto-refresh timer
     */
    function startRefreshTimer() {
        refreshTimer = setInterval(loadDevices, REFRESH_INTERVAL);
    }

    /**
     * Stop auto-refresh timer
     */
    function stopRefreshTimer() {
        if (refreshTimer) {
            clearInterval(refreshTimer);
            refreshTimer = null;
        }
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
