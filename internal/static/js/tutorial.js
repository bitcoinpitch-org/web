/**
 * BitcoinPitch.org Tutorial System
 * Interactive onboarding for first-time visitors
 */

class TutorialEngine {
    constructor() {
        this.currentStep = 0;
        this.isActive = false;
        this.overlay = null;
        this.spotlight = null;
        this.modal = null;
        this.steps = [];
        this.storageKey = 'bitcoinpitch_tutorial_completed';
        
        this.initializeSteps();
        this.bindEvents();
    }

    initializeSteps() {
        // Get current language and translations
        let currentLang = 'en';
        let t = null;
        
        // Check if i18n functions are available
        if (typeof getCurrentLanguage === 'function' && typeof getTutorialTranslations === 'function') {
            currentLang = getCurrentLanguage();
            t = getTutorialTranslations(currentLang);
            console.log('[Tutorial] Using i18n system with language:', currentLang);
        } else {
            console.warn('[Tutorial] i18n functions not available, using fallback English content');
            // Fallback English content
            t = {
                welcome: { title: 'Welcome to BitcoinPitch.org!', content: 'This is your platform for sharing and discovering Bitcoin-related pitches. Let\'s take a quick tour to show you around!' },
                categories: { title: 'Categories', content: 'Browse pitches by category: <strong>Bitcoin</strong> for general topics, <strong>Lightning</strong> for Lightning Network, and <strong>Cashu</strong> for Cashu-related content.' },
                pitchCards: { title: 'Pitch Cards', content: 'This is where Bitcoin pitches appear. Each pitch shows the content, author, and tags. You can <strong>share</strong> on social media or click <strong>tags</strong> to filter similar pitches.' },
                voting: { title: 'Voting System', content: 'When pitches are visible, you can use the <strong>▲ upvote</strong> and <strong>▼ downvote</strong> buttons to rate them. The score shows how the community feels about each pitch and helps the best ones rise to the top!' },
                pitchTypes: { title: 'Pitch Types', content: 'Pitches are categorized by length: <strong>One-liner</strong> (30 chars), <strong>SMS</strong> (80 chars), <strong>Tweet</strong> (280 chars), and <strong>Elevator</strong> (1024 chars).' },
                addPitch: { title: 'Add Your Pitch', content: 'Ready to contribute? Click here to add your own Bitcoin pitch. You can write anything from a quick one-liner to a full elevator pitch!' },
                joinCommunity: { title: 'Join the Community', content: 'Create an account to vote on pitches, save your favorites, and contribute your own ideas. We support Trezor, Nostr, Twitter, and email registration.' },
                allSet: { title: 'You\'re All Set!', content: 'That\'s it! You\'re ready to explore Bitcoin pitches. Start by browsing categories, voting on pitches you like, or adding your own. Welcome to the community!' },
                ui: { skipTour: 'Skip Tour', previous: 'Previous', next: 'Next', finish: 'Finish', of: 'of' }
            };
        }
        
        this.steps = [
            {
                target: '.site-title',
                title: t.welcome.title,
                content: t.welcome.content,
                position: 'bottom',
                showSkip: true
            },
            {
                target: '.nav-tabs',
                title: t.categories.title,
                content: t.categories.content,
                position: 'bottom'
            },
            {
                target: 'article.card, .pitch-list',
                title: t.pitchCards.title,
                content: t.pitchCards.content,
                position: 'right'
            },
            {
                target: '.votes, .up, .down',
                title: t.voting.title,
                content: t.voting.content,
                position: 'left'
            },
            {
                target: '.pitch-types',
                title: t.pitchTypes.title,
                content: t.pitchTypes.content,
                position: 'bottom',
                fallbackIfMissing: '.nav-tabs'
            },
            {
                target: 'button[hx-get="/pitch/form"], .button.primary, .add-pitch',
                title: t.addPitch.title,
                content: t.addPitch.content,
                position: 'left'
            },
            {
                target: '.header-auth, .auth-links',
                title: t.joinCommunity.title,
                content: t.joinCommunity.content,
                position: 'bottom-left'
            },
            {
                target: '.site-title',
                title: t.allSet.title,
                content: t.allSet.content,
                position: 'bottom',
                isLast: true
            }
        ];
        
        // Store translations for later use
        this.translations = t;
    }

    bindEvents() {
        // Start tutorial on page load if first visit
        document.addEventListener('DOMContentLoaded', () => {
            console.log('[Tutorial] DOMContentLoaded fired, checking if tutorial should show...');
            const shouldShow = this.shouldShowTutorial();
            console.log('[Tutorial] Should show tutorial:', shouldShow);
            
            if (shouldShow) {
                console.log('[Tutorial] Starting tutorial in 1 second...');
                // Delay slightly to ensure page is fully rendered
                setTimeout(() => this.start(), 1000);
            }
        });
    }

    shouldShowTutorial() {
        // Don't show if user is logged in (look for user-specific elements)
        const userMenu = document.querySelector('.user-menu, .user-profile, [href="/logout"]');
        console.log('[Tutorial] User menu found:', !!userMenu);
        if (userMenu) {
            return false;
        }

        // Don't show if already completed
        const completed = localStorage.getItem(this.storageKey);
        console.log('[Tutorial] Tutorial completed before:', !!completed);
        if (completed) {
            return false;
        }

        // Only show on homepage or category pages
        const path = window.location.pathname;
        console.log('[Tutorial] Current path:', path);
        if (path === '/' || path === '/bitcoin' || path === '/lightning' || path === '/cashu') {
            return true;
        }

        return false;
    }

    start() {
        if (this.isActive) return;
        
        this.isActive = true;
        this.currentStep = 0;
        this.createOverlay();
        this.showStep(this.currentStep);
        
        // Track tutorial start
        this.trackEvent('tutorial_started');
    }

    createOverlay() {
        // Create dark overlay
        this.overlay = document.createElement('div');
        this.overlay.className = 'tutorial-overlay';
        document.body.appendChild(this.overlay);

        // Create spotlight
        this.spotlight = document.createElement('div');
        this.spotlight.className = 'tutorial-spotlight';
        document.body.appendChild(this.spotlight);

        // Create modal
        this.modal = document.createElement('div');
        this.modal.className = 'tutorial-modal';
        this.modal.innerHTML = `
            <div class="tutorial-content">
                <div class="tutorial-header">
                    <h3 class="tutorial-title"></h3>
                    <button class="tutorial-close" aria-label="Close tutorial">&times;</button>
                </div>
                <div class="tutorial-body"></div>
                <div class="tutorial-footer">
                    <div class="tutorial-progress">
                        <span class="tutorial-current">1</span> ${this.translations.ui.of} <span class="tutorial-total">${this.steps.length}</span>
                    </div>
                    <div class="tutorial-buttons">
                        <button class="tutorial-skip">${this.translations.ui.skipTour}</button>
                        <button class="tutorial-prev" disabled>${this.translations.ui.previous}</button>
                        <button class="tutorial-next">${this.translations.ui.next}</button>
                    </div>
                </div>
            </div>
        `;
        document.body.appendChild(this.modal);

        // Bind modal events
        this.modal.querySelector('.tutorial-close').addEventListener('click', () => this.close());
        this.modal.querySelector('.tutorial-skip').addEventListener('click', () => this.skip());
        this.modal.querySelector('.tutorial-prev').addEventListener('click', () => this.previousStep());
        this.modal.querySelector('.tutorial-next').addEventListener('click', () => this.nextStep());

        // Close on overlay click
        this.overlay.addEventListener('click', () => this.close());
        
        // Prevent modal click from closing
        this.modal.addEventListener('click', (e) => e.stopPropagation());

        // Handle keyboard navigation
        document.addEventListener('keydown', this.handleKeydown.bind(this));
    }

    showStep(stepIndex) {
        if (stepIndex >= this.steps.length) {
            this.complete();
            return;
        }

        const step = this.steps[stepIndex];
        let targetElement = document.querySelector(step.target);

        // Use fallback if target not found
        if (!targetElement && step.fallbackIfMissing) {
            targetElement = document.querySelector(step.fallbackIfMissing);
        }

        if (!targetElement) {
            // Skip this step if target not found
            this.nextStep();
            return;
        }

        // Update spotlight position
        this.updateSpotlight(targetElement);

        // Update modal content
        this.updateModal(step, stepIndex);

        // Position modal
        this.positionModal(targetElement, step.position);

        // Scroll target into view
        targetElement.scrollIntoView({ 
            behavior: 'smooth', 
            block: 'center',
            inline: 'center'
        });
    }

    updateSpotlight(element) {
        const rect = element.getBoundingClientRect();
        const padding = 8;

        this.spotlight.style.left = (rect.left - padding) + 'px';
        this.spotlight.style.top = (rect.top - padding) + 'px';
        this.spotlight.style.width = (rect.width + padding * 2) + 'px';
        this.spotlight.style.height = (rect.height + padding * 2) + 'px';
    }

    updateModal(step, stepIndex) {
        this.modal.querySelector('.tutorial-title').textContent = step.title;
        this.modal.querySelector('.tutorial-body').innerHTML = step.content;
        this.modal.querySelector('.tutorial-current').textContent = stepIndex + 1;

        // Update buttons
        const prevBtn = this.modal.querySelector('.tutorial-prev');
        const nextBtn = this.modal.querySelector('.tutorial-next');
        const skipBtn = this.modal.querySelector('.tutorial-skip');

        prevBtn.disabled = stepIndex === 0;
        
        if (step.isLast) {
            nextBtn.textContent = this.translations.ui.finish;
            skipBtn.style.display = 'none';
        } else {
            nextBtn.textContent = this.translations.ui.next;
            skipBtn.style.display = step.showSkip ? 'inline-block' : 'none';
        }
    }

    positionModal(targetElement, position) {
        const rect = targetElement.getBoundingClientRect();
        const modal = this.modal;
        const margin = 20;

        // Reset positioning
        modal.style.top = '';
        modal.style.left = '';
        modal.style.right = '';
        modal.style.bottom = '';
        modal.style.transform = '';

        // Calculate positions
        switch (position) {
            case 'bottom':
                modal.style.top = (rect.bottom + margin) + 'px';
                modal.style.left = '50%';
                modal.style.transform = 'translateX(-50%)';
                break;
                
            case 'top':
                modal.style.bottom = (window.innerHeight - rect.top + margin) + 'px';
                modal.style.left = '50%';
                modal.style.transform = 'translateX(-50%)';
                break;
                
            case 'right':
                modal.style.left = (rect.right + margin) + 'px';
                modal.style.top = (rect.top + rect.height / 2) + 'px';
                modal.style.transform = 'translateY(-50%)';
                break;
                
            case 'left':
                modal.style.right = (window.innerWidth - rect.left + margin) + 'px';
                modal.style.top = (rect.top + rect.height / 2) + 'px';
                modal.style.transform = 'translateY(-50%)';
                break;
                
            case 'bottom-left':
                modal.style.top = (rect.bottom + margin) + 'px';
                modal.style.right = (window.innerWidth - rect.right) + 'px';
                break;
                
            default: // center
                modal.style.top = '50%';
                modal.style.left = '50%';
                modal.style.transform = 'translate(-50%, -50%)';
        }

        // Ensure modal stays within viewport
        this.ensureModalInViewport();
    }

    ensureModalInViewport() {
        const modal = this.modal;
        const rect = modal.getBoundingClientRect();
        const margin = 10;

        if (rect.left < margin) {
            modal.style.left = margin + 'px';
            modal.style.right = 'auto';
            modal.style.transform = modal.style.transform.replace('translateX(-50%)', '');
        }
        
        if (rect.right > window.innerWidth - margin) {
            modal.style.right = margin + 'px';
            modal.style.left = 'auto';
            modal.style.transform = modal.style.transform.replace('translateX(-50%)', '');
        }
        
        if (rect.top < margin) {
            modal.style.top = margin + 'px';
            modal.style.bottom = 'auto';
            modal.style.transform = modal.style.transform.replace('translateY(-50%)', '');
        }
        
        if (rect.bottom > window.innerHeight - margin) {
            modal.style.bottom = margin + 'px';
            modal.style.top = 'auto';
            modal.style.transform = modal.style.transform.replace('translateY(-50%)', '');
        }
    }

    nextStep() {
        this.currentStep++;
        this.showStep(this.currentStep);
    }

    previousStep() {
        if (this.currentStep > 0) {
            this.currentStep--;
            this.showStep(this.currentStep);
        }
    }

    skip() {
        this.trackEvent('tutorial_skipped', { step: this.currentStep + 1 });
        localStorage.setItem(this.storageKey, 'true');
        this.close();
    }

    complete() {
        this.trackEvent('tutorial_completed');
        localStorage.setItem(this.storageKey, 'true');
        this.close();
    }

    close() {
        if (!this.isActive) return;
        
        this.isActive = false;
        
        // Remove elements
        if (this.overlay) {
            this.overlay.remove();
            this.overlay = null;
        }
        
        if (this.spotlight) {
            this.spotlight.remove();
            this.spotlight = null;
        }
        
        if (this.modal) {
            this.modal.remove();
            this.modal = null;
        }

        // Remove keyboard listener
        document.removeEventListener('keydown', this.handleKeydown.bind(this));
    }

    handleKeydown(e) {
        if (!this.isActive) return;

        switch (e.key) {
            case 'Escape':
                this.close();
                break;
            case 'ArrowRight':
            case ' ':
                e.preventDefault();
                this.nextStep();
                break;
            case 'ArrowLeft':
                e.preventDefault();
                this.previousStep();
                break;
        }
    }

    trackEvent(event, data = {}) {
        // Simple analytics tracking - can be enhanced
        console.log('Tutorial Event:', event, data);
        
        // Could integrate with analytics service here
        // Example: gtag('event', event, data);
    }

    // Public API for manual control
    static getInstance() {
        if (!window.tutorialEngine) {
            window.tutorialEngine = new TutorialEngine();
        }
        return window.tutorialEngine;
    }

    static reset() {
        localStorage.removeItem('bitcoinpitch_tutorial_completed');
    }

    static start() {
        TutorialEngine.getInstance().start();
    }
}

// Initialize tutorial system
console.log('[Tutorial] Initializing tutorial system...');
TutorialEngine.getInstance();

// Expose for debugging
window.Tutorial = TutorialEngine;

console.log('[Tutorial] Tutorial system loaded and available as window.Tutorial'); 