/**
 * Tutorial Internationalization
 * Contains tutorial content in multiple languages
 */

const TUTORIAL_TRANSLATIONS = {
    en: {
        welcome: {
            title: 'Welcome to BitcoinPitch.org!',
            content: 'This is your platform for sharing and discovering Bitcoin-related pitches. Let\'s take a quick tour to show you around!'
        },
        categories: {
            title: 'Categories',
            content: 'Browse pitches by category: <strong>Bitcoin</strong> for general topics, <strong>Lightning</strong> for Lightning Network, and <strong>Cashu</strong> for Cashu-related content.'
        },
        pitchCards: {
            title: 'Pitch Cards',
            content: 'This is where Bitcoin pitches appear. Each pitch shows the content, author, and tags. You can <strong>share</strong> on social media or click <strong>tags</strong> to filter similar pitches.'
        },
        voting: {
            title: 'Voting System',
            content: 'When pitches are visible, you can use the <strong>▲ upvote</strong> and <strong>▼ downvote</strong> buttons to rate them. The score shows how the community feels about each pitch and helps the best ones rise to the top!'
        },
        pitchTypes: {
            title: 'Pitch Types',
            content: 'Pitches are categorized by length: <strong>One-liner</strong> (30 chars), <strong>SMS</strong> (80 chars), <strong>Tweet</strong> (280 chars), and <strong>Elevator</strong> (1024 chars).'
        },
        addPitch: {
            title: 'Add Your Pitch',
            content: 'Ready to contribute? Click here to add your own Bitcoin pitch. You can write anything from a quick one-liner to a full elevator pitch!'
        },
        joinCommunity: {
            title: 'Join the Community',
            content: 'Create an account to vote on pitches, save your favorites, and contribute your own ideas. We support Trezor, Nostr, Twitter, and email registration.'
        },
        allSet: {
            title: 'You\'re All Set!',
            content: 'That\'s it! You\'re ready to explore Bitcoin pitches. Start by browsing categories, voting on pitches you like, or adding your own. Welcome to the community!'
        },
        ui: {
            skipTour: 'Skip Tour',
            previous: 'Previous',
            next: 'Next',
            finish: 'Finish',
            of: 'of'
        }
    },
    cs: {
        welcome: {
            title: 'Vítejte na BitcoinPitch.org!',
            content: 'Toto je vaše platforma pro sdílení a objevování Bitcoin pitchů. Pojďme si udělat rychlou prohlídku!'
        },
        categories: {
            title: 'Kategorie',
            content: 'Procházejte pitche podle kategorií: <strong>Bitcoin</strong> pro obecná témata, <strong>Lightning</strong> pro Lightning Network a <strong>Cashu</strong> pro Cashu obsah.'
        },
        pitchCards: {
            title: 'Pitch Karty',
            content: 'Zde se zobrazují Bitcoin pitche. Každý pitch zobrazuje obsah, autora a štítky. Můžete <strong>sdílet</strong> na sociálních sítích nebo kliknout na <strong>štítky</strong> pro filtrování podobných pitchů.'
        },
        voting: {
            title: 'Hlasovací Systém',
            content: 'Když jsou pitche viditelné, můžete použít tlačítka <strong>▲ pozitivní hlas</strong> a <strong>▼ negativní hlas</strong> pro jejich hodnocení. Skóre ukazuje, jak komunita vnímá každý pitch a pomáhá nejlepším dostat se na vrchol!'
        },
        pitchTypes: {
            title: 'Typy Pitchů',
            content: 'Pitche jsou kategorizovány podle délky: <strong>Jednořádkový</strong> (30 znaků), <strong>SMS</strong> (80 znaků), <strong>Tweet</strong> (280 znaků) a <strong>Výtah</strong> (1024 znaků).'
        },
        addPitch: {
            title: 'Přidejte Svůj Pitch',
            content: 'Připraveni přispět? Klikněte zde pro přidání vlastního Bitcoin pitche. Můžete napsat cokoliv od rychlého jednořádkového po celý výtahový pitch!'
        },
        joinCommunity: {
            title: 'Připojte se ke Komunitě',
            content: 'Vytvořte účet pro hlasování o pitchích, uložení oblíbených a přispívání vlastními nápady. Podporujeme Trezor, Nostr, Twitter a email registraci.'
        },
        allSet: {
            title: 'Vše je Připraveno!',
            content: 'To je vše! Jste připraveni objevovat Bitcoin pitche. Začněte procházením kategorií, hlasováním o pitchích které se vám líbí nebo přidáním vlastních. Vítejte v komunitě!'
        },
        ui: {
            skipTour: 'Přeskočit Prohlídku',
            previous: 'Předchozí',
            next: 'Další',
            finish: 'Dokončit',
            of: 'z'
        }
    }
};

// Utility function to get tutorial translations
function getTutorialTranslations(language = 'en') {
    return TUTORIAL_TRANSLATIONS[language] || TUTORIAL_TRANSLATIONS.en;
}

console.log('[Tutorial i18n] Translation system loaded with languages:', Object.keys(TUTORIAL_TRANSLATIONS));

// Utility function to detect current page language
function getCurrentLanguage() {
    // Try to get language from document lang attribute
    const docLang = document.documentElement.lang;
    if (docLang && TUTORIAL_TRANSLATIONS[docLang]) {
        return docLang;
    }
    
    // Try to get language from URL or other indicators
    const path = window.location.pathname;
    if (path.includes('/cs/') || path.includes('?lang=cs')) {
        return 'cs';
    }
    
    // Check for language cookie
    const langCookie = document.cookie.match(/language=([^;]+)/);
    if (langCookie && TUTORIAL_TRANSLATIONS[langCookie[1]]) {
        return langCookie[1];
    }
    
    // Default to English
    return 'en';
} 