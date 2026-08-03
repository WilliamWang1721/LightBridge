  // Onboarding Tour
export default {
    restartTour: 'Restart Onboarding Tour',
    dontShowAgain: "Don't show again",
    dontShowAgainTitle: 'Permanently close onboarding guide',
    confirmDontShow: "Are you sure you don't want to see the onboarding guide again?\n\nYou can restart it anytime from the user menu in the top right corner.",
    confirmExit: 'Are you sure you want to exit the onboarding guide? You can restart it anytime from the top right menu.',
    interactiveHint: 'Press Enter or Click to continue',
    navigation: {
      flipPage: 'Flip Page',
      exit: 'Exit'
    },
    // Admin tour steps
    admin: {
      welcome: {
        title: '👋 Welcome to LightBridge',
        description: '<div style="line-height: 1.8;"><p style="margin-bottom: 16px;">LightBridge is a powerful AI service gateway platform that helps you easily manage and distribute AI services.</p><p style="margin-bottom: 12px;"><b>🎯 Core Features:</b></p><ul style="margin-left: 20px; margin-bottom: 16px;"><li>📦 <b>Group Management</b> - Create service tiers (VIP, Free Trial, etc.)</li><li>🔗 <b>Account Pool</b> - Connect multiple upstream AI service accounts</li><li>🔑 <b>Key Distribution</b> - Generate independent API Keys for users</li><li>💰 <b>Billing Control</b> - Flexible rate and quota management</li></ul><p style="color: #10b981; font-weight: 600;">Let\'s complete the initial setup in 3 minutes →</p></div>',
        nextBtn: 'Start Setup 🚀',
        prevBtn: 'Skip'
      },
      groupManage: {
        title: '📦 Step 1: Group Management',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;"><b>What is a Group?</b></p><p style="margin-bottom: 12px;">Groups are the core concept of LightBridge, like a "service package":</p><ul style="margin-left: 20px; margin-bottom: 12px; font-size: 13px;"><li>🎯 Each group can contain multiple upstream accounts</li><li>💰 Each group has independent billing multiplier</li><li>👥 Can be set as public or exclusive</li></ul><p style="margin-top: 12px; padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 Example:</b> You can create "VIP Premium" (high rate) and "Free Trial" (low rate) groups</p><p style="margin-top: 16px; color: #10b981; font-weight: 600;">👉 Click "Group Management" on the left sidebar</p></div>'
      },
      createGroup: {
        title: '➕ Create New Group',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Let\'s create your first group.</p><p style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>📝 Tip:</b> Recommend creating a test group first to familiarize yourself with the process</p><p style="color: #10b981; font-weight: 600;">👉 Click the "Create Group" button</p></div>'
      },
      groupName: {
        title: '✏️ 1. Group Name',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Give your group an easy-to-identify name.</p><div style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>💡 Naming Suggestions:</b><ul style="margin: 8px 0 0 16px;"><li>"Test Group" - For testing</li><li>"VIP Premium" - High-quality service</li><li>"Free Trial" - Trial version</li></ul></div><p style="font-size: 13px; color: #6b7280;">Click "Next" when done</p></div>',
        nextBtn: 'Next'
      },
      groupMultiplier: {
        title: '💰 2. Rate Multiplier',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Set the billing multiplier to control user charges.</p><div style="padding: 8px 12px; background: #fef3c7; border-left: 3px solid #f59e0b; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>⚙️ Billing Rules:</b><ul style="margin: 8px 0 0 16px;"><li><b>1.0</b> - Original price (cost price)</li><li><b>1.5</b> - User consumes $1, charged $1.5</li><li><b>2.0</b> - User consumes $1, charged $2</li><li><b>0.8</b> - Subsidy mode (loss-making)</li></ul></div><p style="font-size: 13px; color: #6b7280;">Recommend setting test group to 1.0</p></div>',
        nextBtn: 'Next'
      },
      groupExclusive: {
        title: '🔒 3. Exclusive Group (Optional)',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Control group visibility and access permissions.</p><div style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>🔐 Permission Guide:</b><ul style="margin: 8px 0 0 16px;"><li><b>Off</b> - Public group, visible to all users</li><li><b>On</b> - Exclusive group, only for specified users</li></ul></div><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 Use Cases:</b> VIP exclusive, internal testing, special customers</p></div>',
        nextBtn: 'Next'
      },
      groupSubmit: {
        title: '✅ Save Group',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Confirm the information and click create to save the group.</p><p style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>📌 Group Identity:</b> The icon and accent color are for visual identification only and never affect routing.</p><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>📌 Next Step:</b> After creation, we\'ll add upstream accounts to this group. Its actual upstreams and protocols are derived from those accounts.</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 Click "Create" button</p></div>'
      },
      accountManage: {
        title: '🔗 Step 2: Add Account',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;"><b>Great! Group created successfully 🎉</b></p><p style="margin-bottom: 12px;">Now add upstream AI service accounts to enable actual service delivery.</p><div style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>🔑 Account Purpose:</b><ul style="margin: 8px 0 0 16px;"><li>Connect to upstream AI services (Claude, GPT, Gemini, Grok, etc.)</li><li>One group can contain multiple accounts (load balancing)</li><li>Supports OAuth, Session Key, and other authorization methods</li></ul></div><p style="margin-top: 16px; color: #10b981; font-weight: 600;">👉 Click "Account Management" on the left sidebar</p></div>'
      },
      createAccount: {
        title: '➕ Add New Account',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Click the button to start adding your first upstream account.</p><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 Tip:</b> Recommend using OAuth method - more secure and no manual key extraction needed</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 Click "Add Account" button</p></div>'
      },
      accountName: {
        title: '✏️ 1. Account Name',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Set an easy-to-identify name for the account.</p><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 Naming Suggestions:</b> "Claude Main", "GPT Backup 1", "Test Account", etc.</p></div>',
        nextBtn: 'Next'
      },
      accountPlatform: {
        title: '🤖 2. Select Platform',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Choose the actual upstream provider for this account.</p><p style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px;"><b>📌 Routing:</b> Groups do not have a platform type. The account provider, Custom protocol, and relay mode determine which inbound protocols can use this account.</p></div>',
        nextBtn: 'Next'
      },
      accountType: {
        title: '🔐 3. Authorization Method',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Choose the account authorization method.</p><div style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>✅ Recommended: OAuth Method</b><ul style="margin: 8px 0 0 16px;"><li>No manual key extraction needed</li><li>More secure with auto-refresh support</li><li>Works with Claude Code, ChatGPT OAuth, and Grok OAuth</li></ul></div><div style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px;"><b>📌 Session Key Method</b><ul style="margin: 8px 0 0 16px;"><li>Requires manual extraction from browser</li><li>May need periodic updates</li><li>For platforms without OAuth support</li></ul></div></div>',
        nextBtn: 'Next'
      },
      accountPriority: {
        title: '⚖️ 4. Priority (Optional)',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Set the account call priority.</p><div style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>📊 Priority Rules:</b><ul style="margin: 8px 0 0 16px;"><li>Lower number = higher priority</li><li>System uses low-value accounts first</li><li>Same priority = random selection</li></ul></div><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 Use Case:</b> Set main account to lower value, backup accounts to higher value</p></div>',
        nextBtn: 'Next'
      },
      accountGroups: {
        title: '🎯 5. Assign Groups',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;"><b>Key Step!</b> Assign the account to the group you just created.</p><div style="padding: 8px 12px; background: #fee2e2; border-left: 3px solid #ef4444; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>⚠️ Important Reminder:</b><ul style="margin: 8px 0 0 16px;"><li>Must select at least one group</li><li>Unassigned accounts cannot be used</li><li>One account can be assigned to multiple groups</li></ul></div><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 Tip:</b> Select the test group you just created</p></div>',
        nextBtn: 'Next'
      },
      accountSubmit: {
        title: '✅ Save Account',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Confirm the information and click save.</p><div style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>📌 OAuth Flow:</b><ul style="margin: 8px 0 0 16px;"><li>Will redirect to service provider page after clicking save</li><li>Complete login and authorization on provider page</li><li>Auto-return after successful authorization</li></ul></div><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>📌 Next Step:</b> After adding account, we\'ll create an API key</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 Click "Save" button</p></div>'
      },
      keyManage: {
        title: '🔑 Step 3: Generate Key',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;"><b>Congratulations! Account setup complete 🎉</b></p><p style="margin-bottom: 12px;">Final step: generate an API Key to test if the service works properly.</p><div style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>🔑 API Key Purpose:</b><ul style="margin: 8px 0 0 16px;"><li>Credential for calling AI services</li><li>Each key is bound to one group</li><li>Can set quota and expiration</li><li>Supports independent usage statistics</li></ul></div><p style="margin-top: 16px; color: #10b981; font-weight: 600;">👉 Click "API Keys" on the left sidebar</p></div>'
      },
      createKey: {
        title: '➕ Create Key',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Click the button to create your first API Key.</p><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 Tip:</b> Copy and save immediately after creation - key is only shown once</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 Click "Create Key" button</p></div>'
      },
      keyName: {
        title: '✏️ 1. Key Name',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Set an easy-to-manage name for the key.</p><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 Naming Suggestions:</b> "Test Key", "Production", "Mobile", etc.</p></div>',
        nextBtn: 'Next'
      },
      keyGroup: {
        title: '🎯 2. Select Group',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Select the group you just configured.</p><div style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>📌 Group Determines:</b><ul style="margin: 8px 0 0 16px;"><li>Which accounts this key can use</li><li>What billing multiplier applies</li><li>Whether it\'s an exclusive key</li></ul></div><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 Tip:</b> Select the test group you just created</p></div>',
        nextBtn: 'Next'
      },
      keySubmit: {
        title: '🎉 Generate and Copy',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">System will generate a complete API Key after clicking create.</p><div style="padding: 8px 12px; background: #fee2e2; border-left: 3px solid #ef4444; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>⚠️ Important Reminder:</b><ul style="margin: 8px 0 0 16px;"><li>Key is only shown once, copy immediately</li><li>Need to regenerate if lost</li><li>Keep it safe, don\'t share with others</li></ul></div><div style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>🚀 Next Steps:</b><ul style="margin: 8px 0 0 16px;"><li>Copy the generated sk-xxx key</li><li>Use in any OpenAI-compatible client</li><li>Start experiencing AI services!</li></ul></div><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 Click "Create" button</p></div>'
      }
    },
    // User tour steps
    user: {
      welcome: {
        title: '👋 Welcome to LightBridge',
        description: '<div style="line-height: 1.8;"><p style="margin-bottom: 16px;">Hello! Welcome to the LightBridge AI service platform.</p><p style="margin-bottom: 12px;"><b>🎯 Quick Start:</b></p><ul style="margin-left: 20px; margin-bottom: 16px;"><li>🔑 Create API Key</li><li>📋 Copy key to your application</li><li>🚀 Start using AI services</li></ul><p style="color: #10b981; font-weight: 600;">Just 1 minute, let\'s get started →</p></div>',
        nextBtn: 'Start 🚀',
        prevBtn: 'Skip'
      },
      keyManage: {
        title: '🔑 API Key Management',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Manage all your API access keys here.</p><p style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px;"><b>📌 What is an API Key?</b><br/>An API key is your credential for accessing AI services, like a key that allows your application to call AI capabilities.</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 Click to enter key page</p></div>'
      },
      createKey: {
        title: '➕ Create New Key',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Click the button to create your first API key.</p><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 Tip:</b> Key is only shown once after creation, make sure to copy and save</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 Click "Create Key"</p></div>'
      },
      keyName: {
        title: '✏️ Key Name',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Give your key an easy-to-identify name.</p><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>💡 Examples:</b> "My First Key", "For Testing", etc.</p></div>',
        nextBtn: 'Next'
      },
      keyGroup: {
        title: '🎯 Select Group',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Select the service group assigned by the administrator.</p><p style="padding: 8px 12px; background: #eff6ff; border-left: 3px solid #3b82f6; border-radius: 4px; font-size: 13px;"><b>📌 Group Info:</b><br/>Different groups may have different service quality and billing rates, choose according to your needs.</p></div>',
        nextBtn: 'Next'
      },
      keySubmit: {
        title: '🎉 Complete Creation',
        description: '<div style="line-height: 1.7;"><p style="margin-bottom: 12px;">Click to confirm and create your API key.</p><div style="padding: 8px 12px; background: #fee2e2; border-left: 3px solid #ef4444; border-radius: 4px; font-size: 13px; margin-bottom: 12px;"><b>⚠️ Important:</b><ul style="margin: 8px 0 0 16px;"><li>Copy the key (sk-xxx) immediately after creation</li><li>Key is only shown once, need to regenerate if lost</li></ul></div><p style="padding: 8px 12px; background: #f0fdf4; border-left: 3px solid #10b981; border-radius: 4px; font-size: 13px;"><b>🚀 How to Use:</b><br/>Configure the key in any OpenAI-compatible client (like ChatBox, OpenCat, etc.) and start using!</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 Click "Create" button</p></div>'
      }
    },
    personal: {
      welcome: {
        title: '👋 Welcome to LightBridge Personal Mode',
        description: '<div style="line-height: 1.8;"><p style="margin-bottom: 16px;">Personal mode streamlines the UI down to two steps: connect an account, then create a key.</p><p style="margin-bottom: 12px;"><b>🎯 We will walk you through:</b></p><ul style="margin-left: 20px; margin-bottom: 16px;"><li>🌐 Add an AI account</li><li>🔑 Create an API key</li><li>🚀 Start using it</li></ul><p style="color: #10b981; font-weight: 600;">Let\'s go through it together →</p></div>',
        nextBtn: 'Start 🚀',
        prevBtn: 'Skip'
      },
      accountManage: {
        title: '🌐 Account Management',
        description: '<div style="line-height: 1.7;"><p>Manage the AI accounts you connect (Claude, OpenAI, Gemini, Grok).</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 Open the accounts page</p></div>'
      },
      createAccount: {
        title: '➕ Add Account',
        description: '<div style="line-height: 1.7;"><p>Click to add your first AI account.</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 Click "Add Account"</p></div>'
      },
      accountName: {
        title: '✏️ Account Name',
        description: '<div style="line-height: 1.7;"><p>Give the account a recognizable name, e.g. "My Claude".</p></div>',
        nextBtn: 'Next'
      },
      accountPlatform: {
        title: '🧩 Platform Type',
        description: '<div style="line-height: 1.7;"><p>Pick the platform this account belongs to (Anthropic / OpenAI / Gemini / Grok / Antigravity / Custom).</p></div>',
        nextBtn: 'Next'
      },
      accountType: {
        title: '🔌 Add Method',
        description: '<div style="line-height: 1.7;"><p>Choose how to connect credentials (API Key or OAuth). Other advanced options are collapsed in personal mode — defaults are fine.</p></div>',
        nextBtn: 'Next'
      },
      accountGroups: {
        title: '🎯 Group',
        description: '<div style="line-height: 1.7;"><p>Add the account to a group; keys are scheduled to this account through the group.</p></div>',
        nextBtn: 'Next'
      },
      accountSubmit: {
        title: '✅ Save Account',
        description: '<div style="line-height: 1.7;"><p>Confirm to save the account.</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 Click "Save"</p></div>'
      },
      keyManage: {
        title: '🔑 Key Management',
        description: '<div style="line-height: 1.7;"><p>Once the account is connected, create a calling key here.</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 Open the keys page</p></div>'
      },
      createKey: {
        title: '➕ Create Key',
        description: '<div style="line-height: 1.7;"><p>Click to create an API key.</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 Click "Create Key"</p></div>'
      },
      keySubmit: {
        title: '🎉 Done',
        description: '<div style="line-height: 1.7;"><p>After confirming, copy the key immediately (shown only once). Configure it into any OpenAI-compatible client to start using.</p><p style="margin-top: 12px; color: #10b981; font-weight: 600;">👉 Click "Create"</p></div>'
      }
    }
  }
