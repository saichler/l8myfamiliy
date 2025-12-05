/**
 * Marketing page JavaScript
 * Handles mobile menu and smooth scrolling
 */

(function() {
    'use strict';

    const mobileMenuBtn = document.getElementById('mobileMenuBtn');
    const mobileMenu = document.getElementById('mobileMenu');
    const navbar = document.querySelector('.navbar');

    /**
     * Initialize the marketing page
     */
    function init() {
        setupMobileMenu();
        setupSmoothScroll();
        setupNavbarScroll();
        setupModals();
    }

    /**
     * Set up mobile menu toggle
     */
    function setupMobileMenu() {
        if (!mobileMenuBtn || !mobileMenu) return;

        mobileMenuBtn.addEventListener('click', function() {
            mobileMenu.classList.toggle('active');
            mobileMenuBtn.classList.toggle('active');
        });

        // Close menu when clicking a link
        const menuLinks = mobileMenu.querySelectorAll('a');
        menuLinks.forEach(function(link) {
            link.addEventListener('click', function() {
                mobileMenu.classList.remove('active');
                mobileMenuBtn.classList.remove('active');
            });
        });

        // Close menu when clicking outside
        document.addEventListener('click', function(e) {
            if (!mobileMenu.contains(e.target) && !mobileMenuBtn.contains(e.target)) {
                mobileMenu.classList.remove('active');
                mobileMenuBtn.classList.remove('active');
            }
        });
    }

    /**
     * Set up smooth scrolling for anchor links
     */
    function setupSmoothScroll() {
        const anchorLinks = document.querySelectorAll('a[href^="#"]');

        anchorLinks.forEach(function(link) {
            link.addEventListener('click', function(e) {
                const href = this.getAttribute('href');
                if (href === '#') return;

                const target = document.querySelector(href);
                if (target) {
                    e.preventDefault();
                    const navHeight = navbar ? navbar.offsetHeight : 0;
                    const targetPosition = target.getBoundingClientRect().top + window.pageYOffset - navHeight;

                    window.scrollTo({
                        top: targetPosition,
                        behavior: 'smooth'
                    });
                }
            });
        });
    }

    /**
     * Add shadow to navbar on scroll
     */
    function setupNavbarScroll() {
        if (!navbar) return;

        let lastScroll = 0;

        window.addEventListener('scroll', function() {
            const currentScroll = window.pageYOffset;

            if (currentScroll > 50) {
                navbar.style.boxShadow = '0 4px 20px rgba(0, 0, 0, 0.1)';
            } else {
                navbar.style.boxShadow = '0 2px 10px rgba(0, 0, 0, 0.08)';
            }

            lastScroll = currentScroll;
        });
    }

    /**
     * Set up modal functionality
     */
    function setupModals() {
        var aboutModal = document.getElementById('aboutModal');
        var developerModal = document.getElementById('developerModal');

        // Close modal when clicking outside
        window.addEventListener('click', function(event) {
            if (event.target === aboutModal) {
                closeAboutModal();
            }
            if (event.target === developerModal) {
                closeDeveloperModal();
            }
        });

        // Close modal with Escape key
        document.addEventListener('keydown', function(event) {
            if (event.key === 'Escape') {
                closeAboutModal();
                closeDeveloperModal();
            }
        });
    }

    // Modal functions exposed globally
    window.openAboutModal = function() {
        var modal = document.getElementById('aboutModal');
        if (modal) {
            modal.style.display = 'block';
            document.body.style.overflow = 'hidden';
        }
    };

    window.closeAboutModal = function() {
        var modal = document.getElementById('aboutModal');
        if (modal) {
            modal.style.display = 'none';
            document.body.style.overflow = 'auto';
        }
    };

    window.openDeveloperModal = function() {
        var modal = document.getElementById('developerModal');
        if (modal) {
            modal.style.display = 'block';
            document.body.style.overflow = 'hidden';
        }
    };

    window.closeDeveloperModal = function() {
        var modal = document.getElementById('developerModal');
        if (modal) {
            modal.style.display = 'none';
            document.body.style.overflow = 'auto';
        }
    };

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
