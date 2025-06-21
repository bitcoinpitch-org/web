# Initial Pitches Creation Script

This script creates initial pitches for BitcoinPitch.org to populate the platform with relevant content.

## Overview

The script creates pitches for all three categories (Bitcoin, Lightning, Cashu) in both English and Czech languages, covering all length categories (one-liner, SMS, tweet, elevator). All pitches are posted by the nostr user `npub18jmqxe93tawcmwvhkh4zwwpqmk70whvdxcemyd7xkwl7uy3asulst0w8ra`.

## Features

- **Categories**: Bitcoin, Lightning, Cashu
- **Languages**: English (en), Czech (cs)
- **Lengths**: One-liner, SMS, Tweet, Elevator
- **Author Types**: Same as posted by, Custom names for famous quotes
- **Tags**: Relevant tags for each pitch including **audience targeting**
- **Content**: Based on web search results and Bitcoin knowledge

## Content Sources

The pitches include:
- Original Bitcoin-related content
- Famous Bitcoin quotes (with proper attribution)
- Technical explanations
- Community memes and phrases
- Educational content about Bitcoin, Lightning, and Cashu

## Audience Tags

Each pitch includes **audience tags** to help users find content relevant to their target audience:

### Primary Audience Tags:
- **`investors`** - Pitches aimed at investors, VCs, and financial decision makers
- **`developers`** - Technical content for developers and engineers
- **`business`** - Business-focused pitches for executives and decision makers
- **`general`** - Content suitable for the general public
- **`beginners`** - Educational content for people new to Bitcoin
- **`experts`** - Advanced technical content for experienced users
- **`regulators`** - Content relevant to government and policy makers
- **`merchants`** - Pitches for businesses accepting payments

### Audience Tag Examples:
- **"Bitcoin: Digital gold for the digital age."** → `general`, `beginners`
- **"Bitcoin's proof-of-work creates real-world value..."** → `investors`, `experts`
- **"Lightning Network enables instant, low-cost Bitcoin transactions..."** → `merchants`, `business`
- **"Cashu brings true privacy to Bitcoin..."** → `regulators`, `experts`
- **"Bitcoin is a technological tour de force."** (Bill Gates) → `investors`, `business`

## Setup

1. **Update Database Connection**: Edit the `dbConnStr` constant in the script with your actual database credentials:
   ```go
   const dbConnStr = "host=localhost port=5432 user=bitcoinpitch password=your_password dbname=bitcoinpitch sslmode=disable"
   ```

2. **Ensure Dependencies**: Make sure you have the required Go packages:
   ```bash
   go mod tidy
   ```

## Usage

1. **Run the script**:
   ```bash
   cd scripts
   go run create_initial_pitches.go
   ```

2. **Expected Output**:
   ```
   Created nostr user: [user-id]
   Created pitch: Bitcoin: Digital gold for the digital age.
   Created pitch: Bitcoin is the first decentralized digital currency...
   ...
   Initial pitches creation completed!
   ```

## Pitch Categories

### Bitcoin Category
- Digital gold concept
- Decentralization
- Money properties
- Proof-of-work
- Community memes
- Famous quotes

### Lightning Category
- Speed layer concept
- Payment channels
- Micropayments
- Instant transactions
- Layer 2 scaling

### Cashu Category
- Privacy focus
- Chaumian ecash
- Anonymous transactions
- Financial privacy
- Human rights

## Author Attribution

- **Same as Posted by**: Most pitches use this (posted by nostr user)
- **Custom Names**: Famous quotes are attributed to their original authors:
  - Bill Gates
  - Roger Ver
  - Andreas Antonopoulos
  - Elizabeth Stark
  - Bitcoin Community

## Tags

Each pitch includes relevant tags such as:
- **Content tags**: `bitcoin`, `lightning`, `cashu`, `privacy`, `security`, `decentralization`, `payments`, `money`, `technology`, `quote`, `meme`, `education`
- **Audience tags**: `investors`, `developers`, `business`, `general`, `beginners`, `experts`, `regulators`, `merchants`

## Database Schema

The script works with the existing database schema:
- `users` table for the nostr user
- `pitches` table for pitch content
- `tags` table for tag management
- `pitch_tags` table for many-to-many relationships

## Error Handling

The script includes error handling for:
- Database connection issues
- User creation if not exists
- Tag creation and relationships
- Transaction rollback on errors

## Notes

- The script is idempotent - running it multiple times will create duplicate pitches
- Tags are created with proper usage counting
- All pitches start with zero votes
- The nostr user is created if it doesn't exist
- Czech translations are provided for key concepts
- **Audience tags help users filter content by target audience**

## Customization

To add more pitches or modify existing ones:
1. Edit the `pitches` slice in the script
2. Add new `PitchData` structs
3. Follow the same format for consistency
4. Ensure proper character limits for each length category
5. Include appropriate audience tags for better discoverability 