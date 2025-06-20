// Enable HTMX debug logging to troubleshoot modal content issues
htmx.logAll();

// Configure HTMX to automatically include CSRF token in all requests
htmx.on('htmx:configRequest', function(evt) {
    const csrfToken = document.querySelector('meta[name="csrf-token"]')?.getAttribute('content');
    if (csrfToken) {
        evt.detail.headers['X-CSRF-Token'] = csrfToken;
        console.log('[DEBUG] CSRF token added to request:', csrfToken);
    } else {
        console.warn('[WARNING] No CSRF token found in meta tag');
    }
});

// Verify HTMX is loaded and working
if (typeof htmx !== 'undefined') {
    console.log('[DEBUG] HTMX is loaded successfully');
    console.log('[DEBUG] HTMX version:', htmx.version || 'unknown');
} else {
    console.error('[ERROR] HTMX is not loaded!');
}

// Enhanced Modal Handler - HTMX swap detection
document.addEventListener('htmx:afterSwap', function(evt) {
    console.log('[DEBUG] htmx:afterSwap triggered, target:', evt.detail.target.id || evt.detail.target.className);
    
    // Only show delete modal if the swapped content contains confirmation dialog
    if (evt.detail.target.matches('#delete-confirm-modal .modal-content')) {
        // Check if the swapped content is actually a delete confirmation dialog
        const hasConfirmButton = evt.detail.target.querySelector('.delete-confirm-btn');
        if (hasConfirmButton) {
            console.log('[DEBUG] Delete confirmation content swapped, showing modal');
            const modal = document.getElementById('delete-confirm-modal');
            if (modal) {
                console.log('[DEBUG] Showing delete modal');
                modal.classList.add('active');
                modal.style.display = 'flex'; // Force display
            } else {
                console.error('[ERROR] Delete modal element not found');
            }
        } else {
            console.log('[DEBUG] Non-confirmation content swapped, not showing modal');
        }
    }
    
    // Check if content was swapped into pitch form modal  
    if (evt.detail.target.matches('#pitch-form-modal .modal-content')) {
        // Check if the swapped content is actually a pitch form
        const hasForm = evt.detail.target.querySelector('#pitch-form');
        if (hasForm) {
            console.log('[DEBUG] Pitch form content swapped, showing modal');
            const modal = document.getElementById('pitch-form-modal');
            if (modal) {
                console.log('[DEBUG] Showing pitch form modal');
                modal.classList.add('active');
                activatePitchFormModal();
            } else {
                console.error('[ERROR] Pitch form modal element not found');
            }
        } else {
            console.log('[DEBUG] Non-form content swapped, closing modal (successful submit)');
            const modal = document.getElementById('pitch-form-modal');
            if (modal) {
                console.log('[DEBUG] Closing pitch form modal after successful submit');
                modal.classList.remove('active');
            }
        }
    }

    // Auto-hide success message after 3 seconds
    const successMessage = document.getElementById('privacy-success-message');
    if (successMessage) {
        setTimeout(function() {
            successMessage.style.transition = 'opacity 0.5s ease-out, transform 0.5s ease-out';
            successMessage.style.opacity = '0';
            successMessage.style.transform = 'translateY(-10px)';
            setTimeout(function() {
                if (successMessage.parentNode) {
                    successMessage.remove();
                }
            }, 500);
        }, 3000);
    }
    
    // Re-initialize privacy checkbox interactions
    initPrivacyCheckboxes();
});

// Simple click debug logger that definitely runs
document.addEventListener('click', function(evt) {
    if (evt.target.classList.contains('delete-pitch')) {
        console.log('[DEBUG] Delete button clicked - pitch ID:', evt.target.dataset.pitchId);
    }
    if (evt.target.classList.contains('edit-pitch')) {
        console.log('[DEBUG] Edit button clicked - pitch ID:', evt.target.dataset.pitchId);
    }
    if (evt.target.classList.contains('tab')) {
        console.log('[DEBUG] Tab clicked:', evt.target.getAttribute('data-target'));
    }
    // Close modal if clicking on modal background
    if (evt.target.classList.contains('modal')) {
        console.log('[DEBUG] Modal background clicked, closing modal');
        evt.target.classList.remove('active');
    }
    // Close modal if clicking close button
    if (evt.target.classList.contains('close-modal')) {
        console.log('[DEBUG] Close button clicked');
        const modal = evt.target.closest('.modal') || document.getElementById('delete-confirm-modal');
        if (modal) {
            console.log('[DEBUG] Closing modal:', modal.id);
            modal.classList.remove('active');
        }
    }
}, true);

console.log('[DEBUG] Modal handlers and click listeners registered successfully');

/* ------------------------------------------------- */
/* TAB SWITCHER (only for home page)                 */
/* ------------------------------------------------- */
document.addEventListener('DOMContentLoaded', function() {
    const tabs = document.querySelectorAll('.tab');
    const tabContents = document.querySelectorAll('.tab-content');
    
    console.log('[DEBUG] Tab switcher initializing, found tabs:', tabs.length, 'contents:', tabContents.length);

    // Only enable tab switching if we have multiple tab contents (home page)
    if (tabContents.length > 1) {
        console.log('[DEBUG] Multiple tab contents found, enabling tab switching');
        
        tabs.forEach(tab => {
            tab.addEventListener('click', (e) => {
                // Prevent navigation if we're doing tab switching
                if (tab.querySelector('a')) {
                    e.preventDefault();
                }
                
                console.log('[DEBUG] Tab clicked:', tab.getAttribute('data-target'));
                
                // Remove active class from all tabs and contents
                tabs.forEach(t => t.classList.remove('active'));
                tabContents.forEach(c => c.classList.remove('active'));

                // Add active class to clicked tab and corresponding content
                tab.classList.add('active');
                const targetId = tab.getAttribute('data-target');
                const targetContent = document.getElementById(targetId);
                console.log('[DEBUG] Target content element:', targetContent);
                if (targetContent) {
                    targetContent.classList.add('active');
                }

                // Update URL hash
                window.location.hash = targetId;
                console.log('[DEBUG] Tab switch complete, active tab:', targetId);
            });
        });

        // Handle initial tab based on URL hash
        const hash = window.location.hash.slice(1);
        if (hash) {
            const targetTab = document.querySelector(`[data-target="${hash}"]`);
            if (targetTab) {
                targetTab.click();
            }
        }
    } else {
        console.log('[DEBUG] Single tab content found, tabs will use normal navigation');
    }
});

/* ------------------------------------------------- */
/* VOTE + SCORE HANDLER                              */
/* ------------------------------------------------- */
// Global pitch limits - fetched from API
let pitchLimits = {
    one_liner_min: 3,
    one_liner_max: 30,
    sms_max: 80,
    tweet_max: 280,
    elevator_max: 1120
};

// Fetch pitch limits from API
async function fetchPitchLimits() {
    try {
        const response = await fetch('/api/config/pitch-limits');
        if (response.ok) {
            const limits = await response.json();
            pitchLimits = limits;
            console.log('Pitch limits loaded:', pitchLimits);
        } else {
            console.warn('Failed to fetch pitch limits, using defaults');
        }
    } catch (error) {
        console.warn('Error fetching pitch limits, using defaults:', error);
    }
}

// Initialize pitch limits on page load
document.addEventListener('DOMContentLoaded', function() {
    fetchPitchLimits();
});

// Character counter utility
function setCharCounter(element) {
    const counter = element.querySelector('.char-counter');
    if (!counter) return;
    const current = counter.querySelector('.current');
    const category = counter.querySelector('.length-category');
    element.addEventListener('input', function() {
        const length = this.value.length;
        current.textContent = length;
        const cat = calculateCategory(length);
        category.textContent = `(${cat.charAt(0).toUpperCase() + cat.slice(1)})`;
    });
}

// Score display utility
function setScoreDisplay(scoreEl, value) {
    scoreEl.textContent = value;
    scoreEl.className = 'score';
    if (value > 0) scoreEl.classList.add('positive');
    else if (value < 0) scoreEl.classList.add('negative');
}

// Vote update utility
function updateVote(card, deltaUp, deltaDown) {
    const upCount = card.querySelector('.up-count');
    const downCount = card.querySelector('.down-count');
    const score = card.querySelector('.score');
    
    const newUp = parseInt(upCount.textContent) + deltaUp;
    const newDown = parseInt(downCount.textContent) + deltaDown;
    const newScore = newUp - newDown;
    
    upCount.textContent = newUp;
    downCount.textContent = newDown;
    setScoreDisplay(score, newScore);
}

// Existing pitch form functionality
function activatePitchForm() {
    const form = document.querySelector('#pitch-form');
    if (!form) return;
    
    const content = form.querySelector('#content');
    if (!content) return;
    
    setCharCounter(form);
    
    // Author type handling
    const authorTypeInputs = form.querySelectorAll('input[name="author_type"]');
    const authorInputs = form.querySelectorAll('.author-input');
    
    authorTypeInputs.forEach(input => {
        input.addEventListener('change', function() {
            authorInputs.forEach(el => el.style.display = 'none');
            const type = this.value;
            if (type !== 'same' && type !== 'unknown') {
                const authorEl = form.querySelector(`#author-${type}`);
                if (authorEl) authorEl.style.display = 'block';
            }
        });
    });
    
    // Category toggle buttons
    const categoryBtns = form.querySelectorAll('.category-btn');
    const categoryInput = form.querySelector('#main-category');
    
    categoryBtns.forEach(btn => {
        btn.addEventListener('click', function() {
            categoryBtns.forEach(b => b.classList.remove('active'));
            this.classList.add('active');
            if (categoryInput) categoryInput.value = this.dataset.value;
        });
    });
}

// Handle share buttons
function initShareButtons() {
    document.addEventListener('click', function(evt) {
        // Handle pitch card share links (share-twitter, share-nostr, etc.)
        if (evt.target.classList.contains('share-nostr') || 
            evt.target.classList.contains('share-twitter') || 
            evt.target.classList.contains('share-facebook') || 
            evt.target.classList.contains('share-copy')) {
            
            evt.preventDefault();
            const pitchId = evt.target.dataset.pitchId;
            const platform = evt.target.className.split('-')[1]; // Extract platform from class
            const pitchCard = evt.target.closest('.card');
            const pitchContent = pitchCard.querySelector('.body').textContent.trim();
            
            sharePitch(platform, pitchId, pitchContent);
        }
        
        // Handle detailed view share buttons (share-btn with data-platform)
        if (evt.target.classList.contains('share-btn') || 
            evt.target.closest('.share-btn')) {
            
            evt.preventDefault();
            const btn = evt.target.closest('.share-btn') || evt.target;
            const platform = btn.dataset.platform;
            const pitchId = btn.dataset.pitchId;
            
            // Get pitch content from the detailed view
            const pitchContent = document.querySelector('.pitch-content p')?.textContent.trim() || '';
            
            sharePitch(platform, pitchId, pitchContent);
        }
    });
}

function sharePitch(platform, pitchId, content) {
    const baseUrl = window.location.origin;
    const shareUrl = `${baseUrl}/p/${pitchId}`;
    const attribution = "— bitcoinpitch.org";
    
    switch (platform) {
        case 'twitter':
            const twitterText = encodeURIComponent(`${content}\n\n${shareUrl}\n\n${attribution}`);
            const twitterUrl = `https://twitter.com/intent/tweet?text=${twitterText}`;
            window.open(twitterUrl, '_blank', 'width=550,height=420');
            break;
            
        case 'facebook':
            // Facebook's quote parameter is unreliable, so let's try a different approach
            // Create a shareable text that users can manually paste
            const shareText = `${content}\n\n${shareUrl}\n\n${attribution}`;
            
            if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
                // For localhost, open Facebook and copy content to clipboard as fallback
                console.log('Facebook sharing on localhost');
                console.log('Share text:', shareText);
                
                // Copy to clipboard first
                copyToClipboard(shareText);
                
                // Then open Facebook
                const facebookUrl = `https://www.facebook.com/sharer/sharer.php?u=${encodeURIComponent(shareUrl)}`;
                console.log('Facebook URL:', facebookUrl);
                
                // Show notification that content was copied
                showNotification('Content copied to clipboard! Paste it in the Facebook post.');
                
                // Also show an alert to make it very clear
                setTimeout(() => {
                    alert('📋 Content copied to clipboard!\n\n👉 In the Facebook window that opens, click in the text area and paste (Ctrl+V or Cmd+V) to add your pitch content.');
                }, 500);
                
                window.open(facebookUrl, '_blank', 'width=600,height=400');
            } else {
                // For production, Facebook will scrape Open Graph meta tags from the shared URL
                const facebookUrl = `https://www.facebook.com/sharer/sharer.php?u=${encodeURIComponent(shareUrl)}`;
                console.log('Facebook URL (production):', facebookUrl);
                window.open(facebookUrl, '_blank', 'width=600,height=400');
            }
            break;
            
        case 'nostr':
            const nostrText = `${content}\n\n${shareUrl}\n\n${attribution}`;
            if (window.nostr) {
                // Use WebLN if available
                window.nostr.signEvent({
                    kind: 1,
                    content: nostrText,
                    created_at: Math.floor(Date.now() / 1000),
                    tags: []
                }).then(() => {
                    showNotification('Shared to Nostr!', 'success');
                }).catch(() => {
                    // Fallback to copy
                    copyToClipboard(nostrText);
                    showNotification('Nostr extension not available. Text copied to clipboard.', 'info');
                });
            } else {
                // Fallback to copy
                copyToClipboard(nostrText);
                showNotification('Nostr extension not available. Text copied to clipboard.', 'info');
            }
            break;
            
        case 'copy':
            const copyText = `${content}\n\n${shareUrl}\n\n${attribution}`;
            copyToClipboard(copyText);
            showNotification('Copied to clipboard!', 'success');
            break;
    }
}

function copyToClipboard(text) {
    if (navigator.clipboard && window.isSecureContext) {
        navigator.clipboard.writeText(text);
    } else {
        // Fallback for older browsers
        const textArea = document.createElement('textarea');
        textArea.value = text;
        textArea.style.position = 'fixed';
        textArea.style.left = '-999999px';
        textArea.style.top = '-999999px';
        document.body.appendChild(textArea);
        textArea.focus();
        textArea.select();
        document.execCommand('copy');
        textArea.remove();
    }
}

function showNotification(message, type = 'info') {
    // Create notification element
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: ${type === 'success' ? '#10b981' : type === 'error' ? '#ef4444' : '#3b82f6'};
        color: white;
        padding: 12px 20px;
        border-radius: 6px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        z-index: 1000;
        font-weight: 500;
        transition: all 0.3s ease;
        transform: translateX(100%);
    `;
    
    document.body.appendChild(notification);
    
    // Animate in
    setTimeout(() => {
        notification.style.transform = 'translateX(0)';
    }, 100);
    
    // Auto remove after 3 seconds
    setTimeout(() => {
        notification.style.transform = 'translateX(100%)';
        setTimeout(() => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 300);
    }, 3000);
}

// Initialize share buttons
initShareButtons();

// Handle filters
const filters = document.querySelectorAll('.filter-group select, .filter-group input');
filters.forEach(filter => {
    filter.addEventListener('change', function() {
        // In a real app, this would update the URL and fetch new results
        console.log('Filter changed:', {
            name: this.name,
            value: this.value
        });
    });
});

// Add Pitch Modal
document.addEventListener('DOMContentLoaded', function() {
    const addPitchBtn = document.querySelector('.add-pitch');
    const modal = document.getElementById('pitch-form-modal');
    if (!modal) return;
    const closeBtns = modal.querySelectorAll('.close-modal');
    const form = modal.querySelector('#pitch-form');
    if (!form) return;

    addPitchBtn.addEventListener('click', () => {
        modal.classList.add('active');
    });

    closeBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            modal.classList.remove('active');
        });
    });

    window.addEventListener('click', (e) => {
        if (e.target === modal) {
            modal.classList.remove('active');
        }
    });
});

function activatePitchFormModal() {
    console.log('activatePitchFormModal called');
    const modal = document.getElementById('pitch-form-modal');
    if (!modal) return;
    const form = modal.querySelector('#pitch-form');
    if (!form) return;
    const content = form.querySelector('#content');
    const charCounter = form.querySelector('.char-counter');
    const authorTypeInputs = form.querySelectorAll('input[name="author_type"]');
    const authorInputs = form.querySelectorAll('.author-input');
    // Tag elements are handled by template script

    // Character counter and length category
    content.addEventListener('input', function() {
        const length = this.value.length;
        const max = pitchLimits.elevator_max;
        const submitBtn = form.querySelector('#submit-pitch-btn');
        
        charCounter.querySelector('.current').textContent = length;
        charCounter.querySelector('.max').textContent = max;
        const category = calculateCategory(length);
        charCounter.querySelector('.length-category').textContent = `(${category.charAt(0).toUpperCase() + category.slice(1)})`;
        this.dataset.lengthCategory = category;
        
        // Add warning if over max and disable submit
        if (length > max) {
            charCounter.classList.add('over');
            content.classList.add('over');
            if (submitBtn) {
                submitBtn.disabled = true;
                submitBtn.style.opacity = '0.5';
                submitBtn.style.cursor = 'not-allowed';
            }
        } else {
            charCounter.classList.remove('over');
            content.classList.remove('over');
            if (submitBtn) {
                submitBtn.disabled = false;
                submitBtn.style.opacity = '1';
                submitBtn.style.cursor = 'pointer';
            }
        }
        updateMirror();
    });

    // Author type handling
    authorTypeInputs.forEach(input => {
        input.addEventListener('change', function() {
            authorInputs.forEach(el => el.style.display = 'none');
            const type = this.value;
            if (type !== 'same' && type !== 'unknown') {
                form.querySelector(`#author-${type}`).style.display = 'block';
            }
        });
    });

    // Note: Tag handling is done in the template script, not here

    // Modal close buttons
    modal.querySelectorAll('.close-modal').forEach(btn => {
        btn.onclick = () => modal.classList.remove('active');
    });
    // Close modal on outside click
    modal.onclick = (e) => {
        if (e.target === modal) modal.classList.remove('active');
    };

    // Category toggle buttons
    const categoryBtns = form.querySelectorAll('.category-btn');
    const categoryInput = form.querySelector('#main-category');
    categoryBtns.forEach(btn => {
        btn.addEventListener('click', function() {
            categoryBtns.forEach(b => b.classList.remove('active'));
            this.classList.add('active');
            categoryInput.value = this.dataset.value;
        });
    });

    // Textarea mirror for overflow highlighting
    const mirror = form.querySelector('#content-mirror');
    function updateMirror() {
        if (!mirror) return;
        
        const value = content.value;
        const utf8Length = new TextEncoder().encode(value).length;
        const category = calculateCategory(utf8Length);
        const maxLength = getMaxLengthForCategory(category);
        
        // Calculate overflow based on UTF-8 byte length
        if (utf8Length <= maxLength) {
            // No overflow - clear the mirror
            mirror.innerHTML = '';
        } else {
            // Find the character boundary where we exceed maxLength UTF-8 bytes
            let charIndex = 0;
            let byteCount = 0;
            const encoder = new TextEncoder();
            
            while (charIndex < value.length && byteCount < maxLength) {
                const char = value[charIndex];
                const charBytes = encoder.encode(char).length;
                if (byteCount + charBytes > maxLength) break;
                byteCount += charBytes;
                charIndex++;
            }
            
            const normalPart = value.slice(0, charIndex);
            const overflowPart = value.slice(charIndex);
            
            // Create properly formatted mirror content
            const normalFormatted = normalPart
                .replace(/&/g, '&amp;')
                .replace(/</g, '&lt;')
                .replace(/>/g, '&gt;')
                .replace(/\n/g, '<br>')
                .replace(/ /g, '&nbsp;');
            
            const overflowFormatted = overflowPart
                .replace(/&/g, '&amp;')
                .replace(/</g, '&lt;')
                .replace(/>/g, '&gt;')
                .replace(/\n/g, '<br>')
                .replace(/ /g, '&nbsp;');
            
            // Set mirror content with normal text in transparent color and overflow in red
            mirror.innerHTML = 
                '<span style="color: transparent;">' + normalFormatted + '</span>' + 
                (overflowFormatted ? '<span class="overflow">' + overflowFormatted + '</span>' : '');
        }
        
        // Sync height
        const targetHeight = Math.max(content.scrollHeight, 120);
        mirror.style.height = targetHeight + 'px';
        content.style.height = targetHeight + 'px';
        
        // Sync scroll
        mirror.scrollTop = content.scrollTop;
    }
    content.addEventListener('scroll', function() {
        mirror.scrollTop = content.scrollTop;
    });
    // Initial sync
    updateMirror();
}

// Add debug logging for edit button clicks
document.addEventListener('click', function(evt) {
    if (evt.target.classList.contains('edit-pitch')) {
        console.log('[DEBUG] Edit button clicked:', evt.target.dataset.pitchId);
    }
    if (evt.target.classList.contains('delete-pitch')) {
        console.log('[DEBUG] Delete button clicked:', evt.target.dataset.pitchId, 'hx-get:', evt.target.getAttribute('hx-get'));
        console.log('[DEBUG] Delete button element:', evt.target);
        console.log('[DEBUG] HTMX processing element:', evt.target.hasAttribute('hx-get'));
    }
    if (evt.target.classList.contains('tab')) {
        console.log('[DEBUG] Tab element clicked:', evt.target.getAttribute('data-target'));
    }
});

// Add debug logging for HTMX events
document.addEventListener('htmx:beforeRequest', function(evt) {
    if (evt.detail.elt.classList.contains('edit-pitch')) {
        console.log('[DEBUG] HTMX beforeRequest for edit:', evt.detail.path);
    }
});

document.addEventListener('htmx:afterRequest', function(evt) {
    if (evt.detail.elt.classList.contains('edit-pitch')) {
        console.log('[DEBUG] HTMX afterRequest for edit:', evt.detail.path, 'status:', evt.detail.xhr.status);
    }
});

document.addEventListener('htmx:afterSwap', function(evt) {
    // Debug log for all swaps
    console.log('[DEBUG] HTMX afterSwap:', {
        target: evt.detail.target.id || evt.detail.target.className,
        hasForm: !!evt.detail.target.querySelector('form#pitch-form'),
        content: evt.detail.target.innerHTML.substring(0, 100) + '...'
    });

    // Activate edit/add modal if pitch form is present
    if (evt.detail.target.classList.contains('modal-content') &&
        evt.detail.target.querySelector('form#pitch-form')) {
        console.log('[DEBUG] Activating modal for edit form');
        const modal = document.getElementById('pitch-form-modal');
        if (modal) {
            modal.classList.add('active');
            activatePitchFormModal();
        }
    }

    // Activate delete confirmation modal if swap target is delete modal content
    if (evt.detail.target.classList.contains('modal-content') &&
        evt.detail.target.closest('#delete-confirm-modal')) {
        console.log('[DEBUG] Activating modal for delete confirmation');
        const modal = document.getElementById('delete-confirm-modal');
        if (modal) {
            modal.classList.add('active');
        }
    }
});

// Tag suggestions are handled by template script

console.log('Custom script loaded');
document.addEventListener('htmx:configRequest', function(event) {
    console.log('htmx:configRequest fired');
    var match = document.cookie.match(/_csrf=([^;]+)/);
    if (match) {
        event.detail.headers['X-CSRF-Token'] = decodeURIComponent(match[1]);
        console.log('Set X-CSRF-Token:', decodeURIComponent(match[1]));
    }
});

// Handle successful pitch submissions (both add and edit)
document.addEventListener('htmx:afterRequest', function(evt) {
    if (
        evt.detail &&
        evt.detail.xhr &&
        evt.detail.xhr.status >= 200 &&
        evt.detail.xhr.status < 300 &&
        evt.detail.requestConfig &&
        evt.detail.requestConfig.verb === 'post' &&
        (evt.detail.requestConfig.path.includes('/pitch/add') || evt.detail.requestConfig.path.includes('/pitch/') && evt.detail.requestConfig.path.includes('/edit'))
    ) {
        console.log('[DEBUG] Successful pitch submission detected, closing modal');
        
        // Close the modal after successful pitch submission
        const modal = document.getElementById('pitch-form-modal');
        if (modal) {
            modal.classList.remove('active');
            console.log('[DEBUG] Modal closed successfully');
        }
        
        // Determine if this was add or edit
        const isEdit = evt.detail.requestConfig.path.includes('/edit');
        
        if (isEdit) {
            // For edits, just show success message - card is already updated via HTMX swap
            showNotification('Pitch updated successfully', 'success');
        } else {
            // For new pitches, get the selected category and handle navigation
            const form = document.getElementById('pitch-form');
            if (form) {
                const selectedCategory = form.querySelector('#main-category').value;
                const currentPath = window.location.pathname;
                const currentCategory = currentPath.split('/')[1] || 'bitcoin';
                
                // If the selected category is different from current page, redirect
                if (selectedCategory && selectedCategory !== currentCategory) {
                    window.location.href = `/${selectedCategory}`;
                } else {
                    // Same category, reload to show the new pitch
                    window.location.reload();
                }
            }
        }
    }
});

// Global HTMX error handler for modal 404s
// See: https://htmx.org/quirks/ and https://joshkaramuth.com/blog/django-htmx-not-found/
document.body.addEventListener('htmx:beforeOnLoad', function(evt) {
    if (evt.detail.xhr.status === 404) {
        // Check if the target is the delete-confirm modal
        var modalContent = document.querySelector('#delete-confirm-modal .modal-content');
        if (modalContent) {
            // Insert a minimal error fragment
            modalContent.innerHTML = `
                <div class='modal-header'>
                    <h2>Error</h2>
                    <button class='close-modal' aria-label='Close modal'>&times;</button>
                </div>
                <div class='modal-body'>
                    <p>Pitch not found or invalid request.</p>
                </div>
                <div class='form-actions'>
                    <button type='button' class='button secondary close-modal'>Close</button>
                </div>`;
            // Show the modal if not already visible
            var modal = document.getElementById('delete-confirm-modal');
            if (modal && !modal.classList.contains('active')) {
                modal.classList.add('active');
            }
        }
    }
});

// Listen for the custom HX-Trigger event for modal errors
// See: Solution plan in DEBUG_LOG_DELETE_MODAL.md and https://medium.com/@anuj_jaryal/displaying-error-modals-in-htmx-for-a-seamless-user-experience-55cc87add5ea
document.body.addEventListener('htmx-delete-modal-error', function(evt) {
    var modalContent = document.querySelector('#delete-confirm-modal .modal-content');
    if (modalContent && evt.detail && evt.detail.xhr && evt.detail.xhr.response) {
        modalContent.innerHTML = evt.detail.xhr.response;
        var modal = document.getElementById('delete-confirm-modal');
        if (modal && !modal.classList.contains('active')) {
            modal.classList.add('active');
        }
    }
});

// Enhanced login prompt for non-authenticated users
function showLoginPrompt() {
    // Create a temporary modal with enhanced styling
    const modal = document.createElement('div');
    modal.className = 'login-prompt-modal';
    modal.innerHTML = `
        <div class="login-prompt-content">
            <div class="login-prompt-header">
                <h3>🔐 Login Required</h3>
                <button class="close-login-prompt">&times;</button>
            </div>
            <div class="login-prompt-body">
                <p>You need to be logged in to vote on pitches and join the community!</p>
                <div class="login-options">
                    <a href="/auth/login" class="button primary">Login / Register</a>
                </div>
            </div>
        </div>
    `;
    
    document.body.appendChild(modal);
    
    // Close modal handlers
    modal.querySelector('.close-login-prompt').addEventListener('click', () => {
        modal.style.animation = 'fadeOut 0.2s ease-out';
        setTimeout(() => {
            if (document.body.contains(modal)) {
                document.body.removeChild(modal);
            }
        }, 200);
    });
    
    modal.addEventListener('click', (e) => {
        if (e.target === modal) {
            modal.style.animation = 'fadeOut 0.2s ease-out';
            setTimeout(() => {
                if (document.body.contains(modal)) {
                    document.body.removeChild(modal);
                }
            }, 200);
        }
    });
    
    // Auto-remove after 5 seconds with fade out
    setTimeout(() => {
        if (document.body.contains(modal)) {
            modal.style.animation = 'fadeOut 0.2s ease-out';
            setTimeout(() => {
                if (document.body.contains(modal)) {
                    document.body.removeChild(modal);
                }
            }, 200);
        }
    }, 5000);
}

// Enhanced vote button hover feedback with dynamic tooltips
document.addEventListener('htmx:afterSettle', function() {
    // Add enhanced vote button hover feedback
    document.querySelectorAll('.votes button').forEach(button => {
        if (!button.disabled) {
            button.addEventListener('mouseenter', function() {
                const isUpButton = this.classList.contains('up');
                const isDownButton = this.classList.contains('down');
                const hasVotedUp = this.classList.contains('voted-up');
                const hasVotedDown = this.classList.contains('voted-down');
                
                // Update tooltip based on current state
                if (hasVotedUp) {
                    this.title = '🗑️ Click to remove your upvote';
                    this.setAttribute('aria-label', 'Remove your upvote');
                } else if (hasVotedDown) {
                    this.title = '🗑️ Click to remove your downvote';
                    this.setAttribute('aria-label', 'Remove your downvote');
                } else if (isUpButton) {
                    this.title = '👍 Upvote this pitch';
                    this.setAttribute('aria-label', 'Upvote this pitch');
                } else if (isDownButton) {
                    this.title = '👎 Downvote this pitch';
                    this.setAttribute('aria-label', 'Downvote this pitch');
                }
            });
            
            // Add visual feedback on click
            button.addEventListener('click', function() {
                // Add a temporary "clicked" effect
                this.style.transform = 'scale(0.95)';
                setTimeout(() => {
                    this.style.transform = '';
                }, 150);
            });
        } else {
            // Enhanced feedback for disabled buttons
            button.addEventListener('mouseenter', function() {
                this.title = '🔒 Login required to vote - Click to see login options';
                this.setAttribute('aria-label', 'Login required to vote');
            });
        }
    });
    
    // Add score animation feedback
    document.querySelectorAll('.votes .score').forEach(scoreEl => {
        const score = parseInt(scoreEl.textContent);
        if (score > 0) {
            scoreEl.style.animation = 'pulse-green 0.5s ease-out';
        } else if (score < 0) {
            scoreEl.style.animation = 'pulse-red 0.5s ease-out';
        }
        setTimeout(() => {
            scoreEl.style.animation = '';
        }, 500);
    });
});

// Global filter management functions
function addFilter(filterType, filterValue) {
    const currentUrl = new URL(window.location);
    
    if (filterValue === '' || filterValue === null || filterValue === undefined) {
        // Remove filter if value is empty
        currentUrl.searchParams.delete(filterType);
    } else {
        // Add/update filter
        currentUrl.searchParams.set(filterType, filterValue);
    }
    
    window.location.href = currentUrl.toString();
}

function removeFilter(filterType) {
    const currentUrl = new URL(window.location);
    currentUrl.searchParams.delete(filterType);
    window.location.href = currentUrl.toString();
}

function clearAllFilters() {
    const currentUrl = new URL(window.location);
    // Keep only the path (category)
    const pathOnly = currentUrl.pathname;
    window.location.href = pathOnly;
}

// Update tag links to be additive
function makeTagsAdditive() {
    document.querySelectorAll('.clickable-tag').forEach(tag => {
        tag.addEventListener('click', function(e) {
            e.preventDefault();
            // Try data-tag attribute first (for pitch card tags), then fallback to text content
            const tagName = this.dataset.tag || this.textContent.trim();
            if (tagName) {
                addFilter('tag', tagName);
            }
        });
    });
}

// Update length filter links to be additive
function makeLengthFiltersAdditive() {
    document.querySelectorAll('.pitch-types a').forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            const lengthType = this.getAttribute('data-type');
            if (lengthType) {
                addFilter('length', lengthType);
            }
        });
    });
}

// Initialize additive filtering on page load
document.addEventListener('DOMContentLoaded', function() {
    makeTagsAdditive();
    makeLengthFiltersAdditive();
});

// Re-initialize additive filtering after HTMX swaps (when new content is loaded)
document.addEventListener('htmx:afterSwap', function() {
    makeTagsAdditive();
    makeLengthFiltersAdditive();
});

// Set active tab based on current URL
document.addEventListener('DOMContentLoaded', function() {
    // Get current path
    const currentPath = window.location.pathname;
    
    // Remove active class from all tabs
    const tabs = document.querySelectorAll('.nav-tab');
    tabs.forEach(tab => tab.classList.remove('active'));
    
    // Add active class to current tab
    if (currentPath === '/' || currentPath === '/bitcoin') {
        document.querySelector('.nav-tab[href="/bitcoin"]')?.classList.add('active');
    } else if (currentPath === '/lightning') {
        document.querySelector('.nav-tab[href="/lightning"]')?.classList.add('active');
    } else if (currentPath === '/cashu') {
        document.querySelector('.nav-tab[href="/cashu"]')?.classList.add('active');
    }
});

// Language picker functionality
function toggleLanguageDropdown() {
    const dropdown = document.getElementById('language-dropdown');
    if (dropdown) {
        dropdown.classList.toggle('show');
    }
}

document.addEventListener('DOMContentLoaded', function() {
    // Close language dropdown when clicking outside
    document.addEventListener('click', function(e) {
        if (!e.target.closest('.language-picker')) {
            const dropdown = document.getElementById('language-dropdown');
            if (dropdown) {
                dropdown.classList.remove('show');
            }
        }
    });
    
    // Handle language switching
    const languageOptions = document.querySelectorAll('.language-option');
    languageOptions.forEach(option => {
        option.addEventListener('click', function(e) {
            e.preventDefault();
            const langCode = this.getAttribute('href').split('/').pop();
            
            // Get current page URL to redirect back after language change
            const currentUrl = new URL(window.location);
            const returnUrl = encodeURIComponent(currentUrl.pathname + currentUrl.search);
            
            // Navigate to language switch endpoint with return URL
            window.location.href = `/lang/${langCode}?return=${returnUrl}`;
        });
    });
});

// Category calculation function using dynamic limits - FIXED to match backend logic
function calculateCategory(length) {
    // Match the backend CalculateLengthCategory logic exactly
    if (length >= pitchLimits.one_liner_min && length <= pitchLimits.one_liner_max) {
        return 'one-liner';
    } else if (length <= pitchLimits.sms_max) {
        return 'sms';
    } else if (length <= pitchLimits.tweet_max) {
        return 'tweet';
    } else if (length <= pitchLimits.elevator_max) {
        return 'elevator';
    } else {
        return 'too-long';
    }
}

function getMaxLengthForCategory(category) {
    if (!pitchLimits) return 1024;
    
    switch(category) {
        case 'one-liner': return pitchLimits.one_liner_max;
        case 'sms': return pitchLimits.sms_max;
        case 'tweet': return pitchLimits.tweet_max;
        case 'elevator': return pitchLimits.elevator_max;
        case 'too-long': return pitchLimits.elevator_max;
        default: return pitchLimits.elevator_max;
    }
}

// Update character counting functions to use dynamic limits
function updateCharacterCount() {
    const content = document.getElementById('content');
    if (!content) return;
    
    // Use UTF-8 byte length calculation to match Go's len() function exactly
    const utf8Length = new TextEncoder().encode(content.value.trim()).length;
    
    // DEBUG: Character count investigation
    console.log('[DEBUG] Frontend raw content length:', content.value.length);
    console.log('[DEBUG] Frontend trimmed content length:', content.value.trim().length);
    console.log('[DEBUG] Frontend UTF-8 byte length:', utf8Length);
    console.log('[DEBUG] Frontend content first 50 chars:', content.value.substring(0, 50));
    console.log('[DEBUG] Frontend content last 50 chars:', content.value.substring(content.value.length - 50));
    
    const length = utf8Length;
    const category = calculateCategory(length);
    const maxLength = getMaxLengthForCategory(category);
    
    // Update character count display
    const charCount = document.getElementById('char-count');
    if (charCount) {
        charCount.textContent = `${length}/${maxLength} characters`;
        charCount.className = length > maxLength ? 'char-count over-limit' : 'char-count';
    }
    
    // Update category display  
    const categoryDisplay = document.getElementById('category-display');
    if (categoryDisplay) {
        categoryDisplay.textContent = category.charAt(0).toUpperCase() + category.slice(1);
        categoryDisplay.className = length > maxLength ? 'over-limit' : '';
    }
    
    // Update submit button state
    const submitButton = form.querySelector('button[type="submit"]');
    if (submitButton) {
        submitButton.disabled = length > maxLength || length === 0;
    }
    
    // Update mirror for overflow highlighting
    updateMirror();
}

/* ------------------------------------------------- */
/* DELETE MODAL FUNCTIONS (Global scope for onclick) */
/* ------------------------------------------------- */
window.closeDeleteModal = function() {
    console.log('[DELETE] Closing delete modal');
    const modal = document.getElementById('delete-confirm-modal');
    if (modal) {
        modal.classList.remove('active');
        console.log('[DELETE] Modal closed, active class removed');
    } else {
        console.log('[DELETE] Modal element not found');
    }
};

window.confirmDelete = function(pitchId, csrfToken) {
    console.log('[DELETE] Confirming delete for pitch:', pitchId);
    const button = event.target;
    button.disabled = true;
    button.textContent = 'Deleting...';
    
    fetch(`/pitch/${pitchId}/delete`, {
        method: 'POST',
        headers: {
            'X-CSRF-Token': csrfToken,
            'Content-Type': 'application/json'
        }
    })
    .then(response => {
        console.log('[DELETE] Response:', response.status);
        if (response.ok) {
            // Success - remove pitch card and close modal
            const pitchCard = document.getElementById(`pitch-${pitchId}`);
            if (pitchCard) {
                pitchCard.remove();
                console.log('[DELETE] Pitch card removed from DOM');
            }
            window.closeDeleteModal();
            if (typeof showNotification === 'function') {
                showNotification('Pitch deleted successfully', 'success');
            }
        } else if (response.status === 404) {
            // Pitch not found - show error in modal
            window.showDeleteError('Pitch not found or already deleted');
        } else {
            // Other error - show error in modal  
            window.showDeleteError('Failed to delete pitch: Server error');
        }
    })
    .catch(error => {
        console.error('[DELETE] Error:', error);
        window.showDeleteError('Failed to delete pitch: Network error');
    });
};

window.showDeleteError = function(message) {
    console.log('[DELETE] Showing error:', message);
    const modal = document.getElementById('delete-confirm-modal');
    if (modal) {
        const modalContent = modal.querySelector('.modal-content');
        if (modalContent) {
            modalContent.innerHTML = `
                <div class="modal-header">
                    <h2>Error</h2>
                    <button class="close-modal" aria-label="Close modal" onclick="closeDeleteModal()">&times;</button>
                </div>
                <div class="modal-body">
                    <p>${message}</p>
                </div>
                <div class="form-actions">
                    <button type="button" class="button secondary" onclick="closeDeleteModal()">Close</button>
                </div>
            `;
        }
    }
};

/* ------------------------------------------------- */ 

// Initialize privacy checkbox interactions
function initPrivacyCheckboxes() {
    const privacyCheckboxes = document.querySelectorAll('#privacy-settings-container input[type="checkbox"]');
    
    privacyCheckboxes.forEach(function(checkbox) {
        // Add visual feedback on change
        checkbox.addEventListener('change', function() {
            const option = this.closest('.privacy-option');
            if (option) {
                option.classList.add('loading');
                
                // Remove loading class after a short delay if HTMX fails
                setTimeout(function() {
                    option.classList.remove('loading');
                }, 5000);
            }
        });
        
        // Remove loading class when HTMX request completes
        checkbox.addEventListener('htmx:afterRequest', function() {
            const option = this.closest('.privacy-option');
            if (option) {
                option.classList.remove('loading');
            }
        });
    });
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
    initPrivacyCheckboxes();
});

/* ------------------------------------------------- */ 