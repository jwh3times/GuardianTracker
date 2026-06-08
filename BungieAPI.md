# Bungie.net API Reference

> Complete reference for the Bungie.net API (**version 2.21.8**), generated from `BungieAPI.pdf` (the rendered OpenAPI spec at <https://bungie-net.github.io>).

This document is built for fast lookup during development of Guardian Tracker. It mirrors the official spec: every endpoint (path, verb, parameters, OAuth scope, request body, response type) and every entity/enum (fields, types, manifest mappings). Use your editor search to jump to an operation id (e.g. `Destiny2.GetProfile`) or an entity name (e.g. `Destiny.Definitions.DestinyInventoryItemDefinition`).

## Contents

- [Connecting to the API](#connecting-to-the-api)
- [Authentication & OAuth2](#authentication--oauth2)
- [Standard response envelope](#standard-response-envelope)
- [The Destiny Manifest](#the-destiny-manifest)
- [Endpoint index](#endpoint-index)
- [Endpoints](#endpoints)
- [Entity index](#entity-index)
- [Entities (types & enums)](#entities-types--enums)

## Connecting to the API

**API root path:** `https://www.bungie.net/Platform`

All paths in this document are relative to that root. The full URL for an endpoint is the root path followed by the listed `Path`.

### Required headers

| Header | Required | Notes |
| --- | --- | --- |
| `X-API-Key` | Always | Every request requires an API key. Register an app at <https://www.bungie.net/en/Application> to get one. |
| `Authorization: Bearer <token>` | For authenticated endpoints | OAuth2 access token (see below). Required whenever an endpoint lists an OAuth scope. |
| `Content-Type: application/json` | For POST bodies | Request bodies are JSON. |

## Authentication & OAuth2

Endpoints that read private data or perform actions require an OAuth2 access token. See the [OAuth wiki](https://github.com/Bungie-net/api/wiki/OAuth-Documentation).

| — | URL |
| --- | --- |
| Authorization URL | `https://www.bungie.net/en/OAuth/Authorize` |
| Token URL | `https://www.bungie.net/Platform/App/OAuth/token/` |
| Refresh URL | `https://www.bungie.net/Platform/App/OAuth/token/` |

### Scopes

The `Required Scope(s)` listed on each endpoint refer to these. The numeric value is the bit flag in `Applications.ApplicationScopesEnumeration`.

| Scope | Bit | Description |
| --- | --- | --- |
| `ReadBasicUserProfile` | 1 | Read basic user profile information such as the user's handle, avatar icon, etc. |
| `ReadGroups` | 2 | Read Group/Clan Forums, Wall, and Members for groups and clans that the user has joined. |
| `WriteGroups` | 4 | Write Group/Clan Forums, Wall, and Members for groups and clans that the user has joined. |
| `AdminGroups` | 8 | Administer Group/Clan Forums, Wall, and Members for groups and clans that the user is a founder or an administrator. |
| `BnetWrite` | 16 | Create new groups, clans, and forum posts, along with other actions that are reserved for Bungie.net elevated scope: not meant to be used by third party applications. |
| `MoveEquipDestinyItems` | 32 | Move or equip Destiny items |
| `ReadDestinyInventoryAndVault` | 64 | Read Destiny 1 Inventory and Vault contents. For Destiny 2, this scope is needed to read anything regarded as private. This is the only scope a Destiny 2 app needs for read operations against Destiny 2 data such as inventory, vault, currency, vendors, milestones, progression, etc. |
| `ReadUserData` | 128 | Read user data such as who they are web notifications, clan/group memberships, recent activity, muted users. |
| `EditUserData` | 256 | Edit user data such as preferred language, status, motto, avatar selection and theme. |
| `ReadDestinyVendorsAndAdvisors` | 512 | Access vendor and advisor data specific to a user. OBSOLETE. This scope is only used on the Destiny 1 API. |
| `ReadAndApplyTokens` | 1024 | Read offer history and claim and apply tokens for the user. |
| `AdvancedWriteActions` | 2048 | Can perform actions that will result in a prompt to the user via the Destiny app. |
| `PartnerOfferGrant` | 4096 | Can use the partner offer api to claim rewards defined for a partner |
| `DestinyUnlockValueQuery` | 8192 | Allows an app to query sensitive information like unlock flags and values not available through normal methods. |
| `UserPiiRead` | 16384 | Allows an app to query sensitive user PII, most notably email information. |

> For a Destiny 2 collection tracker, the key scopes are `ReadDestinyInventoryAndVault` (read inventory/vault/collections) and `ReadBasicUserProfile`. Item moves/equips need `MoveEquipDestinyItems`.

## Standard response envelope

**Every** endpoint returns its payload wrapped in this object. In this document the "**Response**" field of each endpoint names the type of the inner `Response` property; the surrounding envelope is always the same and is documented here once.

| Field | Type | Description |
| --- | --- | --- |
| `Response` | (varies — see each endpoint) | The actual payload. |
| `ErrorCode` | int32 | `PlatformErrorCodes` value. `1` = Success. Always check this. |
| `ThrottleSeconds` | int32 | Seconds to wait if throttled. |
| `ErrorStatus` | string | String form of the error code. |
| `Message` | string | Human-readable status/error message. |
| `MessageData` | `Mapping<string, string>` | Additional message data. |
| `DetailedErrorTrace` | string | Detailed trace (usually empty). |

A successful call has `ErrorCode == 1` (`Success`). See the `Exceptions.PlatformErrorCodesEnumeration` entity for the full list of error codes.

## The Destiny Manifest

Most Destiny game data (items, collectibles, vendors, activities, etc.) is delivered as static *definitions* via the **Manifest**, not on every request. Live API responses reference definitions by **hash**. To resolve a hash, look it up in the corresponding manifest definition table.

- `GET /Destiny2/Manifest/` returns the manifest metadata, including a path to the world content SQLite database and per-table JSON paths (`jsonWorldComponentContentPaths`).

- `GET /Destiny2/Manifest/{entityType}/{hashIdentifier}/` fetches a single definition by type and hash without downloading the whole manifest.

- In the entity tables below, a type rendered as `uint32 → SomeDefinition` is a **hash** that maps into that manifest definition (the property is flagged "Mapped to Definition" in the spec). Manifest-backed entities are tagged **(Manifest definition)** with their SQLite table name.

## Endpoint index

135 endpoints across 12 tags.

| Tag | Operation | Method | Path |
| --- | --- | --- | --- |
| App | [App.GetApplicationApiUsage](#appgetapplicationapiusage) | GET | `/App/ApiUsage/{applicationId}/` |
| App | [App.GetBungieApplications](#appgetbungieapplications) | GET | `/App/FirstParty/` |
| User | [User.GetBungieNetUserById](#usergetbungienetuserbyid) | GET | `/User/GetBungieNetUserById/{id}/` |
| User | [User.GetSanitizedPlatformDisplayNames](#usergetsanitizedplatformdisplaynames) | GET | `/User/GetSanitizedPlatformDisplayNames/{membershipId}/` |
| User | [User.GetCredentialTypesForTargetAccount](#usergetcredentialtypesfortargetaccount) | GET | `/User/GetCredentialTypesForTargetAccount/{membershipId}/` |
| User | [User.GetAvailableThemes](#usergetavailablethemes) | GET | `/User/GetAvailableThemes/` |
| User | [User.GetMembershipDataById](#usergetmembershipdatabyid) | GET | `/User/GetMembershipsById/{membershipId}/{membershipType}/` |
| User | [User.GetMembershipDataForCurrentUser](#usergetmembershipdataforcurrentuser) | GET | `/User/GetMembershipsForCurrentUser/` |
| User | [User.GetMembershipFromHardLinkedCredential](#usergetmembershipfromhardlinkedcredential) | GET | `/User/GetMembershipFromHardLinkedCredential/{crType}/{credential}/` |
| User | [User.SearchByGlobalNamePrefix](#usersearchbyglobalnameprefix) | GET | `/User/Search/Prefix/{displayNamePrefix}/{page}/` |
| User | [User.SearchByGlobalNamePost](#usersearchbyglobalnamepost) | POST | `/User/Search/GlobalName/{page}/` |
| Content | [Content.GetContentType](#contentgetcontenttype) | GET | `/Content/GetContentType/{type}/` |
| Content | [Content.GetContentById](#contentgetcontentbyid) | GET | `/Content/GetContentById/{id}/{locale}/` |
| Content | [Content.GetContentByTagAndType](#contentgetcontentbytagandtype) | GET | `/Content/GetContentByTagAndType/{tag}/{type}/{locale}/` |
| Content | [Content.SearchContentWithText](#contentsearchcontentwithtext) | GET | `/Content/Search/{locale}/` |
| Content | [Content.SearchContentByTagAndType](#contentsearchcontentbytagandtype) | GET | `/Content/SearchContentByTagAndType/{tag}/{type}/{locale}/` |
| Content | [Content.SearchHelpArticles](#contentsearchhelparticles) | GET | `/Content/SearchHelpArticles/{searchtext}/{size}/` |
| Content | [Content.RssNewsArticles](#contentrssnewsarticles) | GET | `/Content/Rss/NewsArticles/{pageToken}/` |
| Forum | [Forum.GetTopicsPaged](#forumgettopicspaged) | GET | `/Forum/GetTopicsPaged/{page}/{pageSize}/{group}/{sort}/{quickDate}/{categoryFilter}/` |
| Forum | [Forum.GetCoreTopicsPaged](#forumgetcoretopicspaged) | GET | `/Forum/GetCoreTopicsPaged/{page}/{sort}/{quickDate}/{categoryFilter}/` |
| Forum | [Forum.GetPostsThreadedPaged](#forumgetpoststhreadedpaged) | GET | `/Forum/GetPostsThreadedPaged/{parentPostId}/{page}/{pageSize}/{replySize}/{getParentPost}/` |
| Forum | [Forum.GetPostsThreadedPagedFromChild](#forumgetpoststhreadedpagedfromchild) | GET | `/Forum/GetPostsThreadedPagedFromChild/{childPostId}/{page}/{pageSize}/{replySize}/` |
| Forum | [Forum.GetPostAndParent](#forumgetpostandparent) | GET | `/Forum/GetPostAndParent/{childPostId}/` |
| Forum | [Forum.GetPostAndParentAwaitingApproval](#forumgetpostandparentawaitingapproval) | GET | `/Forum/GetPostAndParentAwaitingApproval/{childPostId}/` |
| Forum | [Forum.GetTopicForContent](#forumgettopicforcontent) | GET | `/Forum/GetTopicForContent/{contentId}/` |
| Forum | [Forum.GetForumTagSuggestions](#forumgetforumtagsuggestions) | GET | `/Forum/GetForumTagSuggestions/` |
| Forum | [Forum.GetPoll](#forumgetpoll) | GET | `/Forum/Poll/{topicId}/` |
| Forum | [Forum.GetRecruitmentThreadSummaries](#forumgetrecruitmentthreadsummaries) | POST | `/Forum/Recruit/Summaries/` |
| GroupV2 | [GroupV2.GetAvailableAvatars](#groupv2getavailableavatars) | GET | `/GroupV2/GetAvailableAvatars/` |
| GroupV2 | [GroupV2.GetAvailableThemes](#groupv2getavailablethemes) | GET | `/GroupV2/GetAvailableThemes/` |
| GroupV2 | [GroupV2.GetUserClanInviteSetting](#groupv2getuserclaninvitesetting) | GET | `/GroupV2/GetUserClanInviteSetting/{mType}/` |
| GroupV2 | [GroupV2.GetRecommendedGroups](#groupv2getrecommendedgroups) | POST | `/GroupV2/Recommended/{groupType}/{createDateRange}/` |
| GroupV2 | [GroupV2.GroupSearch](#groupv2groupsearch) | POST | `/GroupV2/Search/` |
| GroupV2 | [GroupV2.GetGroup](#groupv2getgroup) | GET | `/GroupV2/{groupId}/` |
| GroupV2 | [GroupV2.GetGroupByName](#groupv2getgroupbyname) | GET | `/GroupV2/Name/{groupName}/{groupType}/` |
| GroupV2 | [GroupV2.GetGroupByNameV2](#groupv2getgroupbynamev2) | POST | `/GroupV2/NameV2/` |
| GroupV2 | [GroupV2.GetGroupOptionalConversations](#groupv2getgroupoptionalconversations) | GET | `/GroupV2/{groupId}/OptionalConversations/` |
| GroupV2 | [GroupV2.EditGroup](#groupv2editgroup) | POST | `/GroupV2/{groupId}/Edit/` |
| GroupV2 | [GroupV2.EditClanBanner](#groupv2editclanbanner) | POST | `/GroupV2/{groupId}/EditClanBanner/` |
| GroupV2 | [GroupV2.EditFounderOptions](#groupv2editfounderoptions) | POST | `/GroupV2/{groupId}/EditFounderOptions/` |
| GroupV2 | [GroupV2.AddOptionalConversation](#groupv2addoptionalconversation) | POST | `/GroupV2/{groupId}/OptionalConversations/Add/` |
| GroupV2 | [GroupV2.EditOptionalConversation](#groupv2editoptionalconversation) | POST | `/GroupV2/{groupId}/OptionalConversations/Edit/{conversationId}/` |
| GroupV2 | [GroupV2.GetMembersOfGroup](#groupv2getmembersofgroup) | GET | `/GroupV2/{groupId}/Members/` |
| GroupV2 | [GroupV2.GetAdminsAndFounderOfGroup](#groupv2getadminsandfounderofgroup) | GET | `/GroupV2/{groupId}/AdminsAndFounder/` |
| GroupV2 | [GroupV2.EditGroupMembership](#groupv2editgroupmembership) | POST | `/GroupV2/{groupId}/Members/{membershipType}/{membershipId}/SetMembershipType/` |
| GroupV2 | [GroupV2.KickMember](#groupv2kickmember) | POST | `/GroupV2/{groupId}/Members/{membershipType}/{membershipId}/Kick/` |
| GroupV2 | [GroupV2.BanMember](#groupv2banmember) | POST | `/GroupV2/{groupId}/Members/{membershipType}/{membershipId}/Ban/` |
| GroupV2 | [GroupV2.UnbanMember](#groupv2unbanmember) | POST | `/GroupV2/{groupId}/Members/{membershipType}/{membershipId}/Unban/` |
| GroupV2 | [GroupV2.GetBannedMembersOfGroup](#groupv2getbannedmembersofgroup) | GET | `/GroupV2/{groupId}/Banned/` |
| GroupV2 | [GroupV2.GetGroupEditHistory](#groupv2getgroupedithistory) | GET | `/GroupV2/{groupId}/EditHistory/` |
| GroupV2 | [GroupV2.AbdicateFoundership](#groupv2abdicatefoundership) | POST | `/GroupV2/{groupId}/Admin/AbdicateFoundership/{membershipType}/{founderIdNew}/` |
| GroupV2 | [GroupV2.GetPendingMemberships](#groupv2getpendingmemberships) | GET | `/GroupV2/{groupId}/Members/Pending/` |
| GroupV2 | [GroupV2.GetInvitedIndividuals](#groupv2getinvitedindividuals) | GET | `/GroupV2/{groupId}/Members/InvitedIndividuals/` |
| GroupV2 | [GroupV2.ApproveAllPending](#groupv2approveallpending) | POST | `/GroupV2/{groupId}/Members/ApproveAll/` |
| GroupV2 | [GroupV2.DenyAllPending](#groupv2denyallpending) | POST | `/GroupV2/{groupId}/Members/DenyAll/` |
| GroupV2 | [GroupV2.ApprovePendingForList](#groupv2approvependingforlist) | POST | `/GroupV2/{groupId}/Members/ApproveList/` |
| GroupV2 | [GroupV2.ApprovePending](#groupv2approvepending) | POST | `/GroupV2/{groupId}/Members/Approve/{membershipType}/{membershipId}/` |
| GroupV2 | [GroupV2.DenyPendingForList](#groupv2denypendingforlist) | POST | `/GroupV2/{groupId}/Members/DenyList/` |
| GroupV2 | [GroupV2.GetGroupsForMember](#groupv2getgroupsformember) | GET | `/GroupV2/User/{membershipType}/{membershipId}/{filter}/{groupType}/` |
| GroupV2 | [GroupV2.RecoverGroupForFounder](#groupv2recovergroupforfounder) | GET | `/GroupV2/Recover/{membershipType}/{membershipId}/{groupType}/` |
| GroupV2 | [GroupV2.GetPotentialGroupsForMember](#groupv2getpotentialgroupsformember) | GET | `/GroupV2/User/Potential/{membershipType}/{membershipId}/{filter}/{groupType}/` |
| GroupV2 | [GroupV2.IndividualGroupInvite](#groupv2individualgroupinvite) | POST | `/GroupV2/{groupId}/Members/IndividualInvite/{membershipType}/{membershipId}/` |
| GroupV2 | [GroupV2.IndividualGroupInviteCancel](#groupv2individualgroupinvitecancel) | POST | `/GroupV2/{groupId}/Members/IndividualInviteCancel/{membershipType}/{membershipId}/` |
| Tokens | [Tokens.ForceDropsRepair](#tokensforcedropsrepair) | POST | `/Tokens/Partner/ForceDropsRepair/` |
| Tokens | [Tokens.ClaimPartnerOffer](#tokensclaimpartneroffer) | POST | `/Tokens/Partner/ClaimOffer/` |
| Tokens | [Tokens.ApplyMissingPartnerOffersWithoutClaim](#tokensapplymissingpartnerofferswithoutclaim) | POST | `/Tokens/Partner/ApplyMissingOffers/{partnerApplicationId}/{targetBnetMembershipId}/` |
| Tokens | [Tokens.GetPartnerOfferSkuHistory](#tokensgetpartnerofferskuhistory) | GET | `/Tokens/Partner/History/{partnerApplicationId}/{targetBnetMembershipId}/` |
| Tokens | [Tokens.GetPartnerRewardHistory](#tokensgetpartnerrewardhistory) | GET | `/Tokens/Partner/History/{targetBnetMembershipId}/Application/{partnerApplicationId}/` |
| Tokens | [Tokens.GetBungieRewardsForUser](#tokensgetbungierewardsforuser) | GET | `/Tokens/Rewards/GetRewardsForUser/{membershipId}/` |
| Tokens | [Tokens.GetBungieRewardsForPlatformUser](#tokensgetbungierewardsforplatformuser) | GET | `/Tokens/Rewards/GetRewardsForPlatformUser/{membershipId}/{membershipType}/` |
| Tokens | [Tokens.GetBungieRewardsList](#tokensgetbungierewardslist) | GET | `/Tokens/Rewards/BungieRewards/` |
| Destiny2 | [Destiny2.GetDestinyManifest](#destiny2getdestinymanifest) | GET | `/Destiny2/Manifest/` |
| Destiny2 | [Destiny2.GetDestinyEntityDefinition](#destiny2getdestinyentitydefinition) | GET | `/Destiny2/Manifest/{entityType}/{hashIdentifier}/` |
| Destiny2 | [Destiny2.SearchDestinyPlayerByBungieName](#destiny2searchdestinyplayerbybungiename) | POST | `/Destiny2/SearchDestinyPlayerByBungieName/{membershipType}/` |
| Destiny2 | [Destiny2.GetLinkedProfiles](#destiny2getlinkedprofiles) | GET | `/Destiny2/{membershipType}/Profile/{membershipId}/LinkedProfiles/` |
| Destiny2 | [Destiny2.GetProfile](#destiny2getprofile) | GET | `/Destiny2/{membershipType}/Profile/{destinyMembershipId}/` |
| Destiny2 | [Destiny2.GetCharacter](#destiny2getcharacter) | GET | `/Destiny2/{membershipType}/Profile/{destinyMembershipId}/Character/{characterId}/` |
| Destiny2 | [Destiny2.GetClanWeeklyRewardState](#destiny2getclanweeklyrewardstate) | GET | `/Destiny2/Clan/{groupId}/WeeklyRewardState/` |
| Destiny2 | [Destiny2.GetClanBannerSource](#destiny2getclanbannersource) | GET | `/Destiny2/Clan/ClanBannerDictionary/` |
| Destiny2 | [Destiny2.GetItem](#destiny2getitem) | GET | `/Destiny2/{membershipType}/Profile/{destinyMembershipId}/Item/{itemInstanceId}/` |
| Destiny2 | [Destiny2.GetVendors](#destiny2getvendors) | GET | `/Destiny2/{membershipType}/Profile/{destinyMembershipId}/Character/{characterId}/Vendors/` |
| Destiny2 | [Destiny2.GetVendor](#destiny2getvendor) | GET | `/Destiny2/{membershipType}/Profile/{destinyMembershipId}/Character/{characterId}/Vendors/` |
| Destiny2 | [Destiny2.GetPublicVendors](#destiny2getpublicvendors) | GET | `/Destiny2/Vendors/` |
| Destiny2 | [Destiny2.GetCollectibleNodeDetails](#destiny2getcollectiblenodedetails) | GET | `/Destiny2/{membershipType}/Profile/{destinyMembershipId}/Character/{characterId}/Collectibles/` |
| Destiny2 | [Destiny2.TransferItem](#destiny2transferitem) | POST | `/Destiny2/Actions/Items/TransferItem/` |
| Destiny2 | [Destiny2.PullFromPostmaster](#destiny2pullfrompostmaster) | POST | `/Destiny2/Actions/Items/PullFromPostmaster/` |
| Destiny2 | [Destiny2.EquipItem](#destiny2equipitem) | POST | `/Destiny2/Actions/Items/EquipItem/` |
| Destiny2 | [Destiny2.EquipItems](#destiny2equipitems) | POST | `/Destiny2/Actions/Items/EquipItems/` |
| Destiny2 | [Destiny2.EquipLoadout](#destiny2equiploadout) | POST | `/Destiny2/Actions/Loadouts/EquipLoadout/` |
| Destiny2 | [Destiny2.SnapshotLoadout](#destiny2snapshotloadout) | POST | `/Destiny2/Actions/Loadouts/SnapshotLoadout/` |
| Destiny2 | [Destiny2.UpdateLoadoutIdentifiers](#destiny2updateloadoutidentifiers) | POST | `/Destiny2/Actions/Loadouts/UpdateLoadoutIdentifiers/` |
| Destiny2 | [Destiny2.ClearLoadout](#destiny2clearloadout) | POST | `/Destiny2/Actions/Loadouts/ClearLoadout/` |
| Destiny2 | [Destiny2.SetItemLockState](#destiny2setitemlockstate) | POST | `/Destiny2/Actions/Items/SetLockState/` |
| Destiny2 | [Destiny2.SetQuestTrackedState](#destiny2setquesttrackedstate) | POST | `/Destiny2/Actions/Items/SetTrackedState/` |
| Destiny2 | [Destiny2.InsertSocketPlug](#destiny2insertsocketplug) | POST | `/Destiny2/Actions/Items/InsertSocketPlug/` |
| Destiny2 | [Destiny2.InsertSocketPlugFree](#destiny2insertsocketplugfree) | POST | `/Destiny2/Actions/Items/InsertSocketPlugFree/` |
| Destiny2 | [Destiny2.GetPostGameCarnageReport](#destiny2getpostgamecarnagereport) | GET | `/Destiny2/Stats/PostGameCarnageReport/{activityId}/` |
| Destiny2 | [Destiny2.ReportOffensivePostGameCarnageReportPlayer](#destiny2reportoffensivepostgamecarnagereportplayer) | POST | `/Destiny2/Stats/PostGameCarnageReport/{activityId}/Report/` |
| Destiny2 | [Destiny2.GetHistoricalStatsDefinition](#destiny2gethistoricalstatsdefinition) | GET | `/Destiny2/Stats/Definition/` |
| Destiny2 | [Destiny2.GetClanLeaderboards](#destiny2getclanleaderboards) | GET | `/Destiny2/Stats/Leaderboards/Clans/{groupId}/` |
| Destiny2 | [Destiny2.GetClanAggregateStats](#destiny2getclanaggregatestats) | GET | `/Destiny2/Stats/AggregateClanStats/{groupId}/` |
| Destiny2 | [Destiny2.GetLeaderboards](#destiny2getleaderboards) | GET | `/Destiny2/{membershipType}/Account/{destinyMembershipId}/Stats/Leaderboards/` |
| Destiny2 | [Destiny2.GetLeaderboardsForCharacter](#destiny2getleaderboardsforcharacter) | GET | `/Destiny2/Stats/Leaderboards/{membershipType}/{destinyMembershipId}/{characterId}/` |
| Destiny2 | [Destiny2.SearchDestinyEntities](#destiny2searchdestinyentities) | GET | `/Destiny2/Armory/Search/{type}/{searchTerm}/` |
| Destiny2 | [Destiny2.GetHistoricalStats](#destiny2gethistoricalstats) | GET | `/Destiny2/{membershipType}/Account/{destinyMembershipId}/Character/{characterId}/Stats/` |
| Destiny2 | [Destiny2.GetHistoricalStatsForAccount](#destiny2gethistoricalstatsforaccount) | GET | `/Destiny2/{membershipType}/Account/{destinyMembershipId}/Stats/` |
| Destiny2 | [Destiny2.GetActivityHistory](#destiny2getactivityhistory) | GET | `/Destiny2/{membershipType}/Account/{destinyMembershipId}/Character/{characterId}/Stats/` |
| Destiny2 | [Destiny2.GetUniqueWeaponHistory](#destiny2getuniqueweaponhistory) | GET | `/Destiny2/{membershipType}/Account/{destinyMembershipId}/Character/{characterId}/Stats/` |
| Destiny2 | [Destiny2.GetDestinyAggregateActivityStats](#destiny2getdestinyaggregateactivitystats) | GET | `/Destiny2/{membershipType}/Account/{destinyMembershipId}/Character/{characterId}/Stats/` |
| Destiny2 | [Destiny2.GetPublicMilestoneContent](#destiny2getpublicmilestonecontent) | GET | `/Destiny2/Milestones/{milestoneHash}/Content/` |
| Destiny2 | [Destiny2.GetPublicMilestones](#destiny2getpublicmilestones) | GET | `/Destiny2/Milestones/` |
| Destiny2 | [Destiny2.AwaInitializeRequest](#destiny2awainitializerequest) | POST | `/Destiny2/Awa/Initialize/` |
| Destiny2 | [Destiny2.AwaProvideAuthorizationResult](#destiny2awaprovideauthorizationresult) | POST | `/Destiny2/Awa/AwaProvideAuthorizationResult/` |
| Destiny2 | [Destiny2.AwaGetActionToken](#destiny2awagetactiontoken) | GET | `/Destiny2/Awa/GetActionToken/{correlationId}/` |
| CommunityContent | [CommunityContent.GetCommunityContent](#communitycontentgetcommunitycontent) | GET | `/CommunityContent/Get/{sort}/{mediaFilter}/{page}/` |
| Trending | [Trending.GetTrendingCategories](#trendinggettrendingcategories) | GET | `/Trending/Categories/` |
| Trending | [Trending.GetTrendingCategory](#trendinggettrendingcategory) | GET | `/Trending/Categories/{categoryId}/{pageNumber}/` |
| Trending | [Trending.GetTrendingEntryDetail](#trendinggettrendingentrydetail) | GET | `/Trending/Details/{trendingEntryType}/{identifier}/` |
| Fireteam | [Fireteam.GetActivePrivateClanFireteamCount](#fireteamgetactiveprivateclanfireteamcount) | GET | `/Fireteam/Clan/{groupId}/ActiveCount/` |
| Fireteam | [Fireteam.GetAvailableClanFireteams](#fireteamgetavailableclanfireteams) | GET | `/Fireteam/Clan/{groupId}/Available/{platform}/{activityType}/{dateRange}/{slotFilter}/{publicOnly}/` |
| Fireteam | [Fireteam.SearchPublicAvailableClanFireteams](#fireteamsearchpublicavailableclanfireteams) | GET | `/Fireteam/Search/Available/{platform}/{activityType}/{dateRange}/{slotFilter}/{page}/` |
| Fireteam | [Fireteam.GetMyClanFireteams](#fireteamgetmyclanfireteams) | GET | `/Fireteam/Clan/{groupId}/My/{platform}/{includeClosed}/{page}/` |
| Fireteam | [Fireteam.GetClanFireteam](#fireteamgetclanfireteam) | GET | `/Fireteam/Clan/{groupId}/Summary/{fireteamId}/` |
| Social | [Social.GetFriendList](#socialgetfriendlist) | GET | `/Social/Friends/` |
| Social | [Social.GetFriendRequestList](#socialgetfriendrequestlist) | GET | `/Social/Friends/Requests/` |
| Social | [Social.IssueFriendRequest](#socialissuefriendrequest) | POST | `/Social/Friends/Add/{membershipId}/` |
| Social | [Social.AcceptFriendRequest](#socialacceptfriendrequest) | POST | `/Social/Friends/Requests/Accept/{membershipId}/` |
| Social | [Social.DeclineFriendRequest](#socialdeclinefriendrequest) | POST | `/Social/Friends/Requests/Decline/{membershipId}/` |
| Social | [Social.RemoveFriend](#socialremovefriend) | POST | `/Social/Friends/Remove/{membershipId}/` |
| Social | [Social.RemoveFriendRequest](#socialremovefriendrequest) | POST | `/Social/Friends/Requests/Remove/{membershipId}/` |
| Social | [Social.GetPlatformFriendList](#socialgetplatformfriendlist) | GET | `/Social/PlatformFriends/{friendPlatform}/{page}/` |
| Core | [Core.GetAvailableLocales](#coregetavailablelocales) | GET | `/GetAvailableLocales/` |
| Core | [Core.GetCommonSettings](#coregetcommonsettings) | GET | `/Settings/` |
| Core | [Core.GetUserSystemOverrides](#coregetusersystemoverrides) | GET | `/UserSystemOverrides/` |
| Core | [Core.GetGlobalAlerts](#coregetglobalalerts) | GET | `/GlobalAlerts/` |

## Endpoints

### App

#### App.GetApplicationApiUsage

`GET /App/ApiUsage/{applicationId}/`

Get API usage by application for time frame specified. You can go as far back as 30 days ago, and can ask for up to a 48 hour window of time in a single request. You must be authenticated with at least the ReadUserData permission to access this endpoint.

- **OAuth scope(s):** `ReadUserData`
- **Response:** `Applications.ApiUsage`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `applicationId` | `int32` | ID of the application to get usage statistics. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `end` | `date-time` | End time for query. Goes to now if not specified. |
| `start` | `date-time` | Start time for query. Goes to 24 hours ago if not specified. |

#### App.GetBungieApplications

`GET /App/FirstParty/`

Get list of applications created by Bungie.

- **Response:** `array<Applications.Application>`

### User

#### User.GetBungieNetUserById

`GET /User/GetBungieNetUserById/{id}/`

Loads a bungienet user by membership id.

- **Response:** `User.GeneralUser`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `id` | `int64` | The requested Bungie.net membership id. |

#### User.GetSanitizedPlatformDisplayNames

`GET /User/GetSanitizedPlatformDisplayNames/{membershipId}/`

Gets a list of all display names linked to this membership id but sanitized (profanity filtered). Obeys all visibility rules of calling user and is heavily cached.

- **Response:** `object`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `membershipId` | `int64` | The requested membership id to load. |

#### User.GetCredentialTypesForTargetAccount

`GET /User/GetCredentialTypesForTargetAccount/{membershipId}/`

Returns a list of credential types attached to the requested account

- **Response:** `array<User.Models.GetCredentialTypesForAccountResponse>`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `membershipId` | `int64` | The user's membership id |

#### User.GetAvailableThemes

`GET /User/GetAvailableThemes/`

Returns a list of all available user themes.

- **Response:** `array<Config.UserTheme>`

#### User.GetMembershipDataById

`GET /User/GetMembershipsById/{membershipId}/{membershipType}/`

Returns a list of accounts associated with the supplied membership ID and membership type. This will include all linked accounts (even when hidden) if supplied credentials permit it.

- **Response:** `User.UserMembershipData`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `membershipId` | `int64` | The membership ID of the target user. |
| `membershipType` | `int32` | Type of the supplied membership ID. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

#### User.GetMembershipDataForCurrentUser

`GET /User/GetMembershipsForCurrentUser/`

Returns a list of accounts associated with signed in user. This is useful for OAuth implementations that do not give you access to the token response.

- **OAuth scope(s):** `ReadBasicUserProfile`
- **Response:** `User.UserMembershipData`

#### User.GetMembershipFromHardLinkedCredential

`GET /User/GetMembershipFromHardLinkedCredential/{crType}/{credential}/`

Gets any hard linked membership given a credential. Only works for credentials that are public (just SteamID64 right now). Cross Save aware.

- **Response:** `User.HardLinkedUserMembership`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `credential` | `string` | The credential to look up. Must be a valid SteamID64. |
| `crType` | `byte` | The credential type. 'SteamId' is the only valid value at present. The types of credentials the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.CredentialType. |

#### User.SearchByGlobalNamePrefix

`GET /User/Search/Prefix/{displayNamePrefix}/{page}/`

[OBSOLETE] Do not use this to search users, use SearchByGlobalNamePost instead.

- **Response:** `User.UserSearchResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `displayNamePrefix` | `string` | The display name prefix you're looking for. |
| `page` | `int32` | The zero-based page of results you desire. |

#### User.SearchByGlobalNamePost

`POST /User/Search/GlobalName/{page}/`

Given the prefix of a global display name, returns all users who share that name.

- **Request body:** `User.UserSearchPrefixRequest`
- **Response:** `User.UserSearchResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `page` | `int32` | The zero-based page of results you desire. |

### Content

#### Content.GetContentType

`GET /Content/GetContentType/{type}/`

Gets an object describing a particular variant of content.

- **Response:** `Content.Models.ContentTypeDescription`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `type` | `string` | — |

#### Content.GetContentById

`GET /Content/GetContentById/{id}/{locale}/`

Returns a content item referenced by id

- **Response:** `Content.ContentItemPublicContract`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `id` | `int64` | — |
| `locale` | `string` | — |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `head` | `boolean` | false |

#### Content.GetContentByTagAndType

`GET /Content/GetContentByTagAndType/{tag}/{type}/{locale}/`

Returns the newest item that matches a given tag and Content Type.

- **Response:** `Content.ContentItemPublicContract`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `locale` | `string` | — |
| `tag` | `string` | — |
| `type` | `string` | — |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `head` | `boolean` | Not used. |

#### Content.SearchContentWithText

`GET /Content/Search/{locale}/`

Gets content based on querystring information passed in. Provides basic search and text search capabilities.

- **Response:** `SearchResultOfContentItemPublicContract`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `locale` | `string` | — |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `ctype` | `string` | Content type tag: Help, News, etc. Supply multiple ctypes separated by space. |
| `currentpage` | `int32` | Page number for the search results, starting with page 1. |
| `head` | `boolean` | Not used. |
| `searchtext` | `string` | Word or phrase for the search. |
| `source` | `string` | For analytics, hint at the part of the app that triggered the search. Optional. |
| `tag` | `string` | Tag used on the content to be searched. |

#### Content.SearchContentByTagAndType

`GET /Content/SearchContentByTagAndType/{tag}/{type}/{locale}/`

Searches for Content Items that match the given Tag and Content Type.

- **Response:** `SearchResultOfContentItemPublicContract`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `locale` | `string` | — |
| `tag` | `string` | — |
| `type` | `string` | — |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `currentpage` | `int32` | Page number for the search results starting with page 1. |
| `head` | `boolean` | Not used. |
| `itemsperpage` | `int32` | Not used. |

#### Content.SearchHelpArticles

`GET /Content/SearchHelpArticles/{searchtext}/{size}/`

Search for Help Articles.

- **Response:** `object`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `searchtext` | `string` | — |
| `size` | `string` | — |

#### Content.RssNewsArticles

`GET /Content/Rss/NewsArticles/{pageToken}/`

Returns a JSON string response that is the RSS feed for news articles.

- **Response:** `Content.NewsArticleRssResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `pageToken` | `string` | Zero-based pagination token for paging through result sets. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `categoryfilter` | `string` | Optionally filter response to only include news items in a certain category. |
| `includebody` | `boolean` | Optionally include full content body for each news item. |

### Forum

#### Forum.GetTopicsPaged

`GET /Forum/GetTopicsPaged/{page}/{pageSize}/{group}/{sort}/{quickDate}/{categoryFilter}/`

Get topics from any forum.

- **Response:** `Forum.PostSearchResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `categoryFilter` | `int32` | A category filter |
| `group` | `int64` | The group, if any. |
| `page` | `int32` | Zero paged page number |
| `pageSize` | `int32` | Unused |
| `quickDate` | `int32` | A date filter. |
| `sort` | `byte` | The sort mode. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `locales` | `string` | Comma seperated list of locales posts must match to return in the result list. Default 'en' |
| `tagstring` | `string` | The tags to search, if any. |

#### Forum.GetCoreTopicsPaged

`GET /Forum/GetCoreTopicsPaged/{page}/{sort}/{quickDate}/{categoryFilter}/`

Gets a listing of all topics marked as part of the core group.

- **Response:** `Forum.PostSearchResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `categoryFilter` | `int32` | The category filter. |
| `page` | `int32` | Zero base page |
| `quickDate` | `int32` | The date filter. |
| `sort` | `byte` | The sort mode. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `locales` | `string` | Comma seperated list of locales posts must match to return in the result list. Default 'en' |

#### Forum.GetPostsThreadedPaged

`GET /Forum/GetPostsThreadedPaged/{parentPostId}/{page}/{pageSize}/{replySize}/{getParentPost}/{rootThreadMode}/{sortMode}/`

Returns a thread of posts at the given parent, optionally returning replies to those posts as well as the original parent.

- **Response:** `Forum.PostSearchResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `getParentPost` | `boolean` | — |
| `page` | `int32` | — |
| `pageSize` | `int32` | — |
| `parentPostId` | `int64` | — |
| `replySize` | `int32` | — |
| `rootThreadMode` | `boolean` | — |
| `sortMode` | `int32` | — |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `showbanned` | `string` | If this value is not null or empty, banned posts are requested to be returned |

#### Forum.GetPostsThreadedPagedFromChild

`GET /Forum/GetPostsThreadedPagedFromChild/{childPostId}/{page}/{pageSize}/{replySize}/{rootThreadMode}/{sortMode}/`

Returns a thread of posts starting at the topicId of the input childPostId, optionally returning replies to those posts as well as the original parent.

- **Response:** `Forum.PostSearchResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `childPostId` | `int64` | — |
| `page` | `int32` | — |
| `pageSize` | `int32` | — |
| `replySize` | `int32` | — |
| `rootThreadMode` | `boolean` | — |
| `sortMode` | `int32` | — |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `showbanned` | `string` | If this value is not null or empty, banned posts are requested to be returned |

#### Forum.GetPostAndParent

`GET /Forum/GetPostAndParent/{childPostId}/`

Returns the post specified and its immediate parent.

- **Response:** `Forum.PostSearchResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `childPostId` | `int64` | — |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `showbanned` | `string` | If this value is not null or empty, banned posts are requested to be returned |

#### Forum.GetPostAndParentAwaitingApproval

`GET /Forum/GetPostAndParentAwaitingApproval/{childPostId}/`

Returns the post specified and its immediate parent of posts that are awaiting approval.

- **Response:** `Forum.PostSearchResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `childPostId` | `int64` | — |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `showbanned` | `string` | If this value is not null or empty, banned posts are requested to be returned |

#### Forum.GetTopicForContent

`GET /Forum/GetTopicForContent/{contentId}/`

Gets the post Id for the given content item's comments, if it exists.

- **Response:** `int64`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `contentId` | `int64` | — |

#### Forum.GetForumTagSuggestions

`GET /Forum/GetForumTagSuggestions/`

Gets tag suggestions based on partial text entry, matching them with other tags previously used in the forums.

- **Response:** `array<Tags.Models.Contracts.TagResponse>`

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `partialtag` | `string` | The partial tag input to generate suggestions from. |

#### Forum.GetPoll

`GET /Forum/Poll/{topicId}/`

Gets the specified forum poll.

- **Response:** `Forum.PostSearchResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `topicId` | `int64` | The post id of the topic that has the poll. |

#### Forum.GetRecruitmentThreadSummaries

`POST /Forum/Recruit/Summaries/`

Allows the caller to get a list of to 25 recruitment thread summary information objects.

- **Request body:** `array<int64>`
- **Response:** `array<Forum.ForumRecruitmentDetail>`

### GroupV2

#### GroupV2.GetAvailableAvatars

`GET /GroupV2/GetAvailableAvatars/`

Returns a list of all available group avatars for the signed-in user.

- **Response:** `object`

#### GroupV2.GetAvailableThemes

`GET /GroupV2/GetAvailableThemes/`

Returns a list of all available group themes.

- **Response:** `array<Config.GroupTheme>`

#### GroupV2.GetUserClanInviteSetting

`GET /GroupV2/GetUserClanInviteSetting/{mType}/`

Gets the state of the user's clan invite preferences for a particular membership type - true if they wish to be invited to clans, false otherwise.

- **OAuth scope(s):** `ReadUserData`
- **Response:** `boolean`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `mType` | `int32` | The Destiny membership type of the account we wish to access settings. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

#### GroupV2.GetRecommendedGroups

`POST /GroupV2/Recommended/{groupType}/{createDateRange}/`

Gets groups recommended for you based on the groups to whom those you follow belong.

- **OAuth scope(s):** `ReadGroups`
- **Response:** `array<GroupsV2.GroupV2Card>`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `createDateRange` | `int32` | Requested range in which to pull recommended groups |
| `groupType` | `int32` | Type of groups requested |

#### GroupV2.GroupSearch

`POST /GroupV2/Search/`

Search for Groups.

- **Request body:** `GroupsV2.GroupQuery`
- **Response:** `GroupsV2.GroupSearchResponse`

#### GroupV2.GetGroup

`GET /GroupV2/{groupId}/`

Get information about a specific group of the given ID.

- **Response:** `GroupsV2.GroupResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | Requested group's id. |

#### GroupV2.GetGroupByName

`GET /GroupV2/Name/{groupName}/{groupType}/`

Get information about a specific group with the given name and type.

- **Response:** `GroupsV2.GroupResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupName` | `string` | Exact name of the group to find. |
| `groupType` | `int32` | Type of group to find. |

#### GroupV2.GetGroupByNameV2

`POST /GroupV2/NameV2/`

Get information about a specific group with the given name and type. The POST version.

- **Request body:** `GroupsV2.GroupNameSearchRequest`
- **Response:** `GroupsV2.GroupResponse`

#### GroupV2.GetGroupOptionalConversations

`GET /GroupV2/{groupId}/OptionalConversations/`

Gets a list of available optional conversation channels and their settings.

- **Response:** `array<GroupsV2.GroupOptionalConversation>`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | Requested group's id. |

#### GroupV2.EditGroup

`POST /GroupV2/{groupId}/Edit/`

Edit an existing group. You must have suitable permissions in the group to perform this operation. This latest revision will only edit the fields you pass in - pass null for properties you want to leave unaltered.

- **OAuth scope(s):** `AdminGroups`
- **Request body:** `GroupsV2.GroupEditAction`
- **Response:** `int32`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | Group ID of the group to edit. |

#### GroupV2.EditClanBanner

`POST /GroupV2/{groupId}/EditClanBanner/`

Edit an existing group's clan banner. You must have suitable permissions in the group to perform this operation. All fields are required.

- **OAuth scope(s):** `AdminGroups`
- **Request body:** `GroupsV2.ClanBanner`
- **Response:** `int32`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | Group ID of the group to edit. |

#### GroupV2.EditFounderOptions

`POST /GroupV2/{groupId}/EditFounderOptions/`

Edit group options only available to a founder. You must have suitable permissions in the group to perform this operation.

- **OAuth scope(s):** `AdminGroups`
- **Request body:** `GroupsV2.GroupOptionsEditAction`
- **Response:** `int32`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | Group ID of the group to edit. |

#### GroupV2.AddOptionalConversation

`POST /GroupV2/{groupId}/OptionalConversations/Add/`

Add a new optional conversation/chat channel. Requires admin permissions to the group.

- **OAuth scope(s):** `AdminGroups`
- **Request body:** `GroupsV2.GroupOptionalConversationAddRequest`
- **Response:** `int64`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | Group ID of the group to edit. |

#### GroupV2.EditOptionalConversation

`POST /GroupV2/{groupId}/OptionalConversations/Edit/{conversationId}/`

Edit the settings of an optional conversation/chat channel. Requires admin permissions to the group.

- **OAuth scope(s):** `AdminGroups`
- **Request body:** `GroupsV2.GroupOptionalConversationEditRequest`
- **Response:** `int64`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `conversationId` | `int64` | Conversation Id of the channel being edited. |
| `groupId` | `int64` | Group ID of the group to edit. |

#### GroupV2.GetMembersOfGroup

`GET /GroupV2/{groupId}/Members/`

Get the list of members in a given group.

- **Response:** `SearchResultOfGroupMember`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `currentpage` | `int32` | Page number (starting with 1). Each page has a fixed size of 50 items per page. |
| `groupId` | `int64` | The ID of the group. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `memberType` | `int32` | Filter out other member types. Use None for all members. The member levels used by all V2 Groups API. Individual group types use their own mappings in their native storage (general uses BnetDbGroupMemberType and D2 clans use ClanMemberLevel), but they are all translated to this in the runtime api. These runtime values should NEVER be stored anywhere, so the values can be changed as necessary. |
| `nameSearch` | `string` | The name fragment upon which a search should be executed for members with matching display or unique names. |

#### GroupV2.GetAdminsAndFounderOfGroup

`GET /GroupV2/{groupId}/AdminsAndFounder/`

Get the list of members in a given group who are of admin level or higher.

- **Response:** `SearchResultOfGroupMember`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `currentpage` | `int32` | Page number (starting with 1). Each page has a fixed size of 50 items per page. |
| `groupId` | `int64` | The ID of the group. |

#### GroupV2.EditGroupMembership

`POST /GroupV2/{groupId}/Members/{membershipType}/{membershipId}/SetMembershipType/{memberType}/`

Edit the membership type of a given member. You must have suitable permissions in the group to perform this operation.

- **OAuth scope(s):** `AdminGroups`
- **Response:** `int32`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | ID of the group to which the member belongs. |
| `membershipId` | `int64` | Membership ID to modify. |
| `membershipType` | `int32` | Membership type of the provide membership ID. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |
| `memberType` | `int32` | New membertype for the specified member. The member levels used by all V2 Groups API. Individual group types use their own mappings in their native storage (general uses BnetDbGroupMemberType and D2 clans use ClanMemberLevel), but they are all translated to this in the runtime api. These runtime values should NEVER be stored anywhere, so the values can be changed as necessary. |

#### GroupV2.KickMember

`POST /GroupV2/{groupId}/Members/{membershipType}/{membershipId}/Kick/`

Kick a member from the given group, forcing them to reapply if they wish to re-join the group. You must have suitable permissions in the group to perform this operation.

- **OAuth scope(s):** `AdminGroups`
- **Response:** `GroupsV2.GroupMemberLeaveResult`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | Group ID to kick the user from. |
| `membershipId` | `int64` | Membership ID to kick. |
| `membershipType` | `int32` | Membership type of the provided membership ID. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

#### GroupV2.BanMember

`POST /GroupV2/{groupId}/Members/{membershipType}/{membershipId}/Ban/`

Bans the requested member from the requested group for the specified period of time.

- **OAuth scope(s):** `AdminGroups`
- **Request body:** `GroupsV2.GroupBanRequest`
- **Response:** `int32`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | Group ID that has the member to ban. |
| `membershipId` | `int64` | Membership ID of the member to ban from the group. |
| `membershipType` | `int32` | Membership type of the provided membership ID. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

#### GroupV2.UnbanMember

`POST /GroupV2/{groupId}/Members/{membershipType}/{membershipId}/Unban/`

Unbans the requested member, allowing them to re-apply for membership.

- **OAuth scope(s):** `AdminGroups`
- **Response:** `int32`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | — |
| `membershipId` | `int64` | Membership ID of the member to unban from the group |
| `membershipType` | `int32` | Membership type of the provided membership ID. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

#### GroupV2.GetBannedMembersOfGroup

`GET /GroupV2/{groupId}/Banned/`

Get the list of banned members in a given group. Only accessible to group Admins and above. Not applicable to all groups. Check group features.

- **OAuth scope(s):** `AdminGroups`
- **Response:** `SearchResultOfGroupBan`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `currentpage` | `int32` | Page number (starting with 1). Each page has a fixed size of 50 entries. |
| `groupId` | `int64` | Group ID whose banned members you are fetching |

#### GroupV2.GetGroupEditHistory

`GET /GroupV2/{groupId}/EditHistory/`

Get the list of edits made to a given group. Only accessible to group Admins and above.

- **OAuth scope(s):** `AdminGroups`
- **Response:** `SearchResultOfGroupEditHistory`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `currentpage` | `int32` | Page number (starting with 1). Each page has a fixed size of 50 entries. |
| `groupId` | `int64` | Group ID whose edit history you are fetching |

#### GroupV2.AbdicateFoundership

`POST /GroupV2/{groupId}/Admin/AbdicateFoundership/{membershipType}/{founderIdNew}/`

An administrative method to allow the founder of a group or clan to give up their position to another admin permanently.

- **Response:** `boolean`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `founderIdNew` | `int64` | The new founder for this group. Must already be a group admin. |
| `groupId` | `int64` | The target group id. |
| `membershipType` | `int32` | Membership type of the provided founderIdNew. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

#### GroupV2.GetPendingMemberships

`GET /GroupV2/{groupId}/Members/Pending/`

Get the list of users who are awaiting a decision on their application to join a given group. Modified to include application info.

- **OAuth scope(s):** `AdminGroups`
- **Response:** `SearchResultOfGroupMemberApplication`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `currentpage` | `int32` | Page number (starting with 1). Each page has a fixed size of 50 items per page. |
| `groupId` | `int64` | ID of the group. |

#### GroupV2.GetInvitedIndividuals

`GET /GroupV2/{groupId}/Members/InvitedIndividuals/`

Get the list of users who have been invited into the group.

- **OAuth scope(s):** `AdminGroups`
- **Response:** `SearchResultOfGroupMemberApplication`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `currentpage` | `int32` | Page number (starting with 1). Each page has a fixed size of 50 items per page. |
| `groupId` | `int64` | ID of the group. |

#### GroupV2.ApproveAllPending

`POST /GroupV2/{groupId}/Members/ApproveAll/`

Approve all of the pending users for the given group.

- **OAuth scope(s):** `AdminGroups`
- **Request body:** `GroupsV2.GroupApplicationRequest`
- **Response:** `array<Entities.EntityActionResult>`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | ID of the group. |

#### GroupV2.DenyAllPending

`POST /GroupV2/{groupId}/Members/DenyAll/`

Deny all of the pending users for the given group.

- **OAuth scope(s):** `AdminGroups`
- **Request body:** `GroupsV2.GroupApplicationRequest`
- **Response:** `array<Entities.EntityActionResult>`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | ID of the group. |

#### GroupV2.ApprovePendingForList

`POST /GroupV2/{groupId}/Members/ApproveList/`

Approve all of the pending users for the given group.

- **OAuth scope(s):** `AdminGroups`
- **Request body:** `GroupsV2.GroupApplicationListRequest`
- **Response:** `array<Entities.EntityActionResult>`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | ID of the group. |

#### GroupV2.ApprovePending

`POST /GroupV2/{groupId}/Members/Approve/{membershipType}/{membershipId}/`

Approve the given membershipId to join the group/clan as long as they have applied.

- **OAuth scope(s):** `AdminGroups`
- **Request body:** `GroupsV2.GroupApplicationRequest`
- **Response:** `boolean`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | ID of the group. |
| `membershipId` | `int64` | The membership id being approved. |
| `membershipType` | `int32` | Membership type of the supplied membership ID. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

#### GroupV2.DenyPendingForList

`POST /GroupV2/{groupId}/Members/DenyList/`

Deny all of the pending users for the given group that match the passed-in .

- **OAuth scope(s):** `AdminGroups`
- **Request body:** `GroupsV2.GroupApplicationListRequest`
- **Response:** `array<Entities.EntityActionResult>`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | ID of the group. |

#### GroupV2.GetGroupsForMember

`GET /GroupV2/User/{membershipType}/{membershipId}/{filter}/{groupType}/`

Get information about the groups that a given member has joined.

- **Response:** `GroupsV2.GetGroupsForMemberResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `filter` | `int32` | Filter apply to list of joined groups. |
| `groupType` | `int32` | Type of group the supplied member founded. |
| `membershipId` | `int64` | Membership ID to for which to find founded groups. |
| `membershipType` | `int32` | Membership type of the supplied membership ID. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

#### GroupV2.RecoverGroupForFounder

`GET /GroupV2/Recover/{membershipType}/{membershipId}/{groupType}/`

Allows a founder to manually recover a group they can see in game but not on bungie.net

- **Response:** `GroupsV2.GroupMembershipSearchResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupType` | `int32` | Type of group the supplied member founded. |
| `membershipId` | `int64` | Membership ID to for which to find founded groups. |
| `membershipType` | `int32` | Membership type of the supplied membership ID. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

#### GroupV2.GetPotentialGroupsForMember

`GET /GroupV2/User/Potential/{membershipType}/{membershipId}/{filter}/{groupType}/`

Get information about the groups that a given member has applied to or been invited to.

- **Response:** `GroupsV2.GroupPotentialMembershipSearchResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `filter` | `int32` | Filter apply to list of potential joined groups. |
| `groupType` | `int32` | Type of group the supplied member applied. |
| `membershipId` | `int64` | Membership ID to for which to find applied groups. |
| `membershipType` | `int32` | Membership type of the supplied membership ID. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

#### GroupV2.IndividualGroupInvite

`POST /GroupV2/{groupId}/Members/IndividualInvite/{membershipType}/{membershipId}/`

Invite a user to join this group.

- **OAuth scope(s):** `AdminGroups`
- **Request body:** `GroupsV2.GroupApplicationRequest`
- **Response:** `GroupsV2.GroupApplicationResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | ID of the group you would like to join. |
| `membershipId` | `int64` | Membership id of the account being invited. |
| `membershipType` | `int32` | MembershipType of the account being invited. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

#### GroupV2.IndividualGroupInviteCancel

`POST /GroupV2/{groupId}/Members/IndividualInviteCancel/{membershipType}/{membershipId}/`

Cancels a pending invitation to join a group.

- **OAuth scope(s):** `AdminGroups`
- **Response:** `GroupsV2.GroupApplicationResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | ID of the group you would like to join. |
| `membershipId` | `int64` | Membership id of the account being cancelled. |
| `membershipType` | `int32` | MembershipType of the account being cancelled. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

### Tokens

#### Tokens.ForceDropsRepair

`POST /Tokens/Partner/ForceDropsRepair/`

Twitch Drops self-repair function - scans twitch for drops not marked as fulfilled and resyncs them.

- **OAuth scope(s):** `PartnerOfferGrant`
- **Response:** `boolean`

#### Tokens.ClaimPartnerOffer

`POST /Tokens/Partner/ClaimOffer/`

Claim a partner offer as the authenticated user.

- **OAuth scope(s):** `PartnerOfferGrant`
- **Request body:** `Tokens.PartnerOfferClaimRequest`
- **Response:** `boolean`

#### Tokens.ApplyMissingPartnerOffersWithoutClaim

`POST /Tokens/Partner/ApplyMissingOffers/{partnerApplicationId}/{targetBnetMembershipId}/`

Apply a partner offer to the targeted user. This endpoint does not claim a new offer, but any already claimed offers will be applied to the game if not already.

- **OAuth scope(s):** `PartnerOfferGrant`
- **Response:** `boolean`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `partnerApplicationId` | `int32` | The partner application identifier. |
| `targetBnetMembershipId` | `int64` | The bungie.net user to apply missing offers to. If not self, elevated permissions are required. |

#### Tokens.GetPartnerOfferSkuHistory

`GET /Tokens/Partner/History/{partnerApplicationId}/{targetBnetMembershipId}/`

Returns the partner sku and offer history of the targeted user. Elevated permissions are required to see users that are not yourself.

- **OAuth scope(s):** `PartnerOfferGrant`
- **Response:** `array<Tokens.PartnerOfferSkuHistoryResponse>`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `partnerApplicationId` | `int32` | The partner application identifier. |
| `targetBnetMembershipId` | `int64` | The bungie.net user to apply missing offers to. If not self, elevated permissions are required. |

#### Tokens.GetPartnerRewardHistory

`GET /Tokens/Partner/History/{targetBnetMembershipId}/Application/{partnerApplicationId}/`

Returns the partner rewards history of the targeted user, both partner offers and Twitch drops.

- **OAuth scope(s):** `PartnerOfferGrant`
- **Response:** `Tokens.PartnerRewardHistoryResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `partnerApplicationId` | `int32` | The partner application identifier. |
| `targetBnetMembershipId` | `int64` | The bungie.net user to return reward history for. |

#### Tokens.GetBungieRewardsForUser

`GET /Tokens/Rewards/GetRewardsForUser/{membershipId}/`

Returns the bungie rewards for the targeted user.

- **OAuth scope(s):** `ReadAndApplyTokens`
- **Response:** `object`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `membershipId` | `int64` | bungie.net user membershipId for requested user rewards. If not self, elevated permissions are required. |

#### Tokens.GetBungieRewardsForPlatformUser

`GET /Tokens/Rewards/GetRewardsForPlatformUser/{membershipId}/{membershipType}/`

Returns the bungie rewards for the targeted user when a platform membership Id and Type are used.

- **OAuth scope(s):** `ReadAndApplyTokens`
- **Response:** `object`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `membershipId` | `int64` | users platform membershipId for requested user rewards. If not self, elevated permissions are required. |
| `membershipType` | `int32` | The target Destiny 2 membership type. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

#### Tokens.GetBungieRewardsList

`GET /Tokens/Rewards/BungieRewards/`

Returns a list of the current bungie rewards

- **Response:** `object`

### Destiny2

#### Destiny2.GetDestinyManifest

`GET /Destiny2/Manifest/`

Returns the current version of the manifest as a json object.

- **Response:** `Destiny.Config.DestinyManifest`

#### Destiny2.GetDestinyEntityDefinition

`GET /Destiny2/Manifest/{entityType}/{hashIdentifier}/`

Returns the static definition of an entity of the given Type and hash identifier. Examine the API Documentation for the Type Names of entities that have their own definitions. Note that the return type will always *inherit from* DestinyDefinition, but the specific type returned will be the requested entity type if it can be found. Please don't use this as a chatty alternative to the Manifest database if you require large sets of data, but for simple and one-off accesses this should be handy.

- **Response:** `Destiny.Definitions.DestinyDefinition`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `entityType` | `string` | The type of entity for whom you would like results. These correspond to the entity's definition contract name. For instance, if you are looking for items, this property should be 'DestinyInventoryItemDefinition'. PREVIEW: This endpoint is still in beta, and may experience rough edges. The schema is tentatively in final form, but there may be bugs that prevent desirable operation. |
| `hashIdentifier` | `uint32` | The hash identifier for the specific Entity you want returned. |

#### Destiny2.SearchDestinyPlayerByBungieName

`POST /Destiny2/SearchDestinyPlayerByBungieName/{membershipType}/`

Returns a list of Destiny memberships given a global Bungie Display Name. This method will hide overridden memberships due to cross save.

- **Request body:** `User.ExactSearchRequest`
- **Response:** `array<User.UserInfoCard>`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `membershipType` | `int32` | A valid non-BungieNet membership type, or All. Indicates which memberships to return. You probably want this set to All. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

#### Destiny2.GetLinkedProfiles

`GET /Destiny2/{membershipType}/Profile/{membershipId}/LinkedProfiles/`

Returns a summary information about all profiles linked to the requesting membership type/ membership ID that have valid Destiny information. The passed-in Membership Type/Membership ID may be a Bungie.Net membership or a Destiny membership. It only returns the minimal amount of data to begin making more substantive requests, but will hopefully serve as a useful alternative to UserServices for people who just care about Destiny data. Note that it will only return linked accounts whose linkages you are allowed to view.

- **Response:** `Destiny.Responses.DestinyLinkedProfilesResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `membershipId` | `int64` | The ID of the membership whose linked Destiny accounts you want returned. Make sure your membership ID matches its Membership Type: don't pass us a PSN membership ID and the XBox membership type, it's not going to work! |
| `membershipType` | `int32` | The type for the membership whose linked Destiny accounts you want returned. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `getAllMemberships` | `boolean` | (optional) if set to 'true', all memberships regardless of whether they're obscured by overrides will be returned. Normal privacy restrictions on account linking will still apply no matter what. |

#### Destiny2.GetProfile

`GET /Destiny2/{membershipType}/Profile/{destinyMembershipId}/`

Returns Destiny Profile information for the supplied membership.

- **Response:** `Destiny.Responses.DestinyProfileResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `destinyMembershipId` | `int64` | Destiny membership ID. |
| `membershipType` | `int32` | A valid non-BungieNet membership type. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `components` | `array<int32>` | A comma separated list of components to return (as strings or numeric values). See the DestinyComponentType enum for valid components to request. You must request at least one component to receive results. |

#### Destiny2.GetCharacter

`GET /Destiny2/{membershipType}/Profile/{destinyMembershipId}/Character/{characterId}/`

Returns character information for the supplied character.

- **Response:** `Destiny.Responses.DestinyCharacterResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `characterId` | `int64` | ID of the character. |
| `destinyMembershipId` | `int64` | Destiny membership ID. |
| `membershipType` | `int32` | A valid non-BungieNet membership type. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `components` | `array<int32>` | A comma separated list of components to return (as strings or numeric values). See the DestinyComponentType enum for valid components to request. You must request at least one component to receive results. |

#### Destiny2.GetClanWeeklyRewardState

`GET /Destiny2/Clan/{groupId}/WeeklyRewardState/`

Returns information on the weekly clan rewards and if the clan has earned them or not. Note that this will always report rewards as not redeemed.

- **Response:** `Destiny.Milestones.DestinyMilestone`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | A valid group id of clan. |

#### Destiny2.GetClanBannerSource

`GET /Destiny2/Clan/ClanBannerDictionary/`

Returns the dictionary of values for the Clan Banner

- **Response:** `Config.ClanBanner.ClanBannerSource`

#### Destiny2.GetItem

`GET /Destiny2/{membershipType}/Profile/{destinyMembershipId}/Item/{itemInstanceId}/`

Retrieve the details of an instanced Destiny Item. An instanced Destiny item is one with an ItemInstanceId. Non-instanced items, such as materials, have no useful instance-specific details and thus are not queryable here.

- **Response:** `Destiny.Responses.DestinyItemResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `destinyMembershipId` | `int64` | The membership ID of the destiny profile. |
| `itemInstanceId` | `int64` | The Instance ID of the destiny item. |
| `membershipType` | `int32` | A valid non-BungieNet membership type. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `components` | `array<int32>` | A comma separated list of components to return (as strings or numeric values). See the DestinyComponentType enum for valid components to request. You must request at least one component to receive results. |

#### Destiny2.GetVendors

`GET /Destiny2/{membershipType}/Profile/{destinyMembershipId}/Character/{characterId}/Vendors/`

Get currently available vendors from the list of vendors that can possibly have rotating inventory. Note that this does not include things like preview vendors and vendors-as-kiosks, neither of whom have rotating/dynamic inventories. Use their definitions as-is for those.

- **Response:** `Destiny.Responses.DestinyVendorsResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `characterId` | `int64` | The Destiny Character ID of the character for whom we're getting vendor info. |
| `destinyMembershipId` | `int64` | Destiny membership ID of another user. You may be denied. |
| `membershipType` | `int32` | A valid non-BungieNet membership type. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `components` | `array<int32>` | A comma separated list of components to return (as strings or numeric values). See the DestinyComponentType enum for valid components to request. You must request at least one component to receive results. |
| `filter` | `int32` | The filter of what vendors and items to return, if any. Indicates the type of filter to apply to Vendor results. |

#### Destiny2.GetVendor

`GET /Destiny2/{membershipType}/Profile/{destinyMembershipId}/Character/{characterId}/Vendors/{vendorHash}/`

Get the details of a specific Vendor.

- **Response:** `Destiny.Responses.DestinyVendorResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `characterId` | `int64` | The Destiny Character ID of the character for whom we're getting vendor info. |
| `destinyMembershipId` | `int64` | Destiny membership ID of another user. You may be denied. |
| `membershipType` | `int32` | A valid non-BungieNet membership type. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |
| `vendorHash` | `uint32` | The Hash identifier of the Vendor to be returned. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `components` | `array<int32>` | A comma separated list of components to return (as strings or numeric values). See the DestinyComponentType enum for valid components to request. You must request at least one component to receive results. |

#### Destiny2.GetPublicVendors

`GET /Destiny2/Vendors/`

Get items available from vendors where the vendors have items for sale that are common for everyone. If any portion of the Vendor's available inventory is character or account specific, we will be unable to return their data from this endpoint due to the way that available inventory is computed. As I am often guilty of saying: 'It's a long story...'

- **Response:** `Destiny.Responses.DestinyPublicVendorsResponse`

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `components` | `array<int32>` | A comma separated list of components to return (as strings or numeric values). See the DestinyComponentType enum for valid components to request. You must request at least one component to receive results. |

#### Destiny2.GetCollectibleNodeDetails

`GET /Destiny2/{membershipType}/Profile/{destinyMembershipId}/Character/{characterId}/Collectibles/{collectiblePresentationNodeHash}/`

Given a Presentation Node that has Collectibles as direct descendants, this will return item details about those descendants in the context of the requesting character.

- **Response:** `Destiny.Responses.DestinyCollectibleNodeDetailResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `characterId` | `int64` | The Destiny Character ID of the character for whom we're getting collectible detail info. |
| `collectiblePresentationNodeHash` | `uint32` | The hash identifier of the Presentation Node for whom we should return collectible details. Details will only be returned for collectibles that are direct descendants of this node. |
| `destinyMembershipId` | `int64` | Destiny membership ID of another user. You may be denied. |
| `membershipType` | `int32` | A valid non-BungieNet membership type. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `components` | `array<int32>` | A comma separated list of components to return (as strings or numeric values). See the DestinyComponentType enum for valid components to request. You must request at least one component to receive results. |

#### Destiny2.TransferItem

`POST /Destiny2/Actions/Items/TransferItem/`

Transfer an item to/from your vault. You must have a valid Destiny account. You must also pass BOTH a reference AND an instance ID if it's an instanced item. itshappening.gif

- **OAuth scope(s):** `MoveEquipDestinyItems`
- **Request body:** `Destiny.Requests.DestinyItemTransferRequest`
- **Response:** `int32`

#### Destiny2.PullFromPostmaster

`POST /Destiny2/Actions/Items/PullFromPostmaster/`

Extract an item from the Postmaster, with whatever implications that may entail. You must have a valid Destiny account. You must also pass BOTH a reference AND an instance ID if it's an instanced item.

- **OAuth scope(s):** `MoveEquipDestinyItems`
- **Request body:** `Destiny.Requests.Actions.DestinyPostmasterTransferRequest`
- **Response:** `int32`

#### Destiny2.EquipItem

`POST /Destiny2/Actions/Items/EquipItem/`

Equip an item. You must have a valid Destiny Account, and either be in a social space, in orbit, or offline.

- **OAuth scope(s):** `MoveEquipDestinyItems`
- **Request body:** `Destiny.Requests.Actions.DestinyItemActionRequest`
- **Response:** `int32`

#### Destiny2.EquipItems

`POST /Destiny2/Actions/Items/EquipItems/`

Equip a list of items by itemInstanceIds. You must have a valid Destiny Account, and either be in a social space, in orbit, or offline. Any items not found on your character will be ignored.

- **OAuth scope(s):** `MoveEquipDestinyItems`
- **Request body:** `Destiny.Requests.Actions.DestinyItemSetActionRequest`
- **Response:** `Destiny.DestinyEquipItemResults`

#### Destiny2.EquipLoadout

`POST /Destiny2/Actions/Loadouts/EquipLoadout/`

Equip a loadout. You must have a valid Destiny Account, and either be in a social space, in orbit, or offline.

- **OAuth scope(s):** `MoveEquipDestinyItems`
- **Request body:** `Destiny.Requests.Actions.DestinyLoadoutActionRequest`
- **Response:** `int32`

#### Destiny2.SnapshotLoadout

`POST /Destiny2/Actions/Loadouts/SnapshotLoadout/`

Snapshot a loadout with the currently equipped items.

- **OAuth scope(s):** `MoveEquipDestinyItems`
- **Request body:** `Destiny.Requests.Actions.DestinyLoadoutUpdateActionRequest`
- **Response:** `int32`

#### Destiny2.UpdateLoadoutIdentifiers

`POST /Destiny2/Actions/Loadouts/UpdateLoadoutIdentifiers/`

Update the color, icon, and name of a loadout.

- **OAuth scope(s):** `MoveEquipDestinyItems`
- **Request body:** `Destiny.Requests.Actions.DestinyLoadoutUpdateActionRequest`
- **Response:** `int32`

#### Destiny2.ClearLoadout

`POST /Destiny2/Actions/Loadouts/ClearLoadout/`

Clear the identifiers and items of a loadout.

- **OAuth scope(s):** `MoveEquipDestinyItems`
- **Request body:** `Destiny.Requests.Actions.DestinyLoadoutActionRequest`
- **Response:** `int32`

#### Destiny2.SetItemLockState

`POST /Destiny2/Actions/Items/SetLockState/`

Set the Lock State for an instanced item. You must have a valid Destiny Account.

- **OAuth scope(s):** `MoveEquipDestinyItems`
- **Request body:** `Destiny.Requests.Actions.DestinyItemStateRequest`
- **Response:** `int32`

#### Destiny2.SetQuestTrackedState

`POST /Destiny2/Actions/Items/SetTrackedState/`

Set the Tracking State for an instanced item, if that item is a Quest or Bounty. You must have a valid Destiny Account. Yeah, it's an item.

- **OAuth scope(s):** `MoveEquipDestinyItems`
- **Request body:** `Destiny.Requests.Actions.DestinyItemStateRequest`
- **Response:** `int32`

#### Destiny2.InsertSocketPlug

`POST /Destiny2/Actions/Items/InsertSocketPlug/`

Insert a plug into a socketed item. I know how it sounds, but I assure you it's much more G-rated than you might be guessing. We haven't decided yet whether this will be able to insert plugs that have side effects, but if we do it will require special scope permission for an application attempting to do so. You must have a valid Destiny Account, and either be in a social space, in orbit, or offline. Request must include proof of permission for 'InsertPlugs' from the account owner.

- **OAuth scope(s):** `AdvancedWriteActions`
- **Request body:** `Destiny.Requests.Actions.DestinyInsertPlugsActionRequest`
- **Response:** `Destiny.Responses.DestinyItemChangeResponse`

#### Destiny2.InsertSocketPlugFree

`POST /Destiny2/Actions/Items/InsertSocketPlugFree/`

Insert a 'free' plug into an item's socket. This does not require 'Advanced Write Action' authorization and is available to 3rd-party apps, but will only work on 'free and reversible' socket actions (Perks, Armor Mods, Shaders, Ornaments, etc.). You must have a valid Destiny Account, and the character must either be in a social space, in orbit, or offline.

- **OAuth scope(s):** `MoveEquipDestinyItems`
- **Request body:** `Destiny.Requests.Actions.DestinyInsertPlugsFreeActionRequest`
- **Response:** `Destiny.Responses.DestinyItemChangeResponse`

#### Destiny2.GetPostGameCarnageReport

`GET /Destiny2/Stats/PostGameCarnageReport/{activityId}/`

Gets the available post game carnage report for the activity ID.

- **Response:** `Destiny.HistoricalStats.DestinyPostGameCarnageReportData`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `activityId` | `int64` | The ID of the activity whose PGCR is requested. |

#### Destiny2.ReportOffensivePostGameCarnageReportPlayer

`POST /Destiny2/Stats/PostGameCarnageReport/{activityId}/Report/`

Report a player that you met in an activity that was engaging in ToS-violating activities. Both you and the offending player must have played in the activityId passed in. Please use this judiciously and only when you have strong suspicions of violation, pretty please.

- **OAuth scope(s):** `BnetWrite`
- **Request body:** `Destiny.Reporting.Requests.DestinyReportOffensePgcrRequest`
- **Response:** `int32`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `activityId` | `int64` | The ID of the activity where you ran into the brigand that you're reporting. |

#### Destiny2.GetHistoricalStatsDefinition

`GET /Destiny2/Stats/Definition/`

Gets historical stats definitions.

- **Response:** `object`

#### Destiny2.GetClanLeaderboards

`GET /Destiny2/Stats/Leaderboards/Clans/{groupId}/`

Gets leaderboards with the signed in user's friends and the supplied destinyMembershipId as the focus. PREVIEW: This endpoint is still in beta, and may experience rough edges. The schema is in final form, but there may be bugs that prevent desirable operation.

- **Response:** `object`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | Group ID of the clan whose leaderboards you wish to fetch. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `maxtop` | `int32` | Maximum number of top players to return. Use a large number to get entire leaderboard. |
| `modes` | `string` | List of game modes for which to get leaderboards. See the documentation for DestinyActivityModeType for valid values, and pass in string representation, comma delimited. |
| `statid` | `string` | ID of stat to return rather than returning all Leaderboard stats. |

#### Destiny2.GetClanAggregateStats

`GET /Destiny2/Stats/AggregateClanStats/{groupId}/`

Gets aggregated stats for a clan using the same categories as the clan leaderboards. PREVIEW: This endpoint is still in beta, and may experience rough edges. The schema is in final form, but there may be bugs that prevent desirable operation.

- **Response:** `array<Destiny.HistoricalStats.DestinyClanAggregateStat>`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | Group ID of the clan whose leaderboards you wish to fetch. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `modes` | `string` | List of game modes for which to get leaderboards. See the documentation for DestinyActivityModeType for valid values, and pass in string representation, comma delimited. |

#### Destiny2.GetLeaderboards

`GET /Destiny2/{membershipType}/Account/{destinyMembershipId}/Stats/Leaderboards/`

Gets leaderboards with the signed in user's friends and the supplied destinyMembershipId as the focus. PREVIEW: This endpoint has not yet been implemented. It is being returned for a preview of future functionality, and for public comment/suggestion/preparation.

- **Response:** `object`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `destinyMembershipId` | `int64` | The Destiny membershipId of the user to retrieve. |
| `membershipType` | `int32` | A valid non-BungieNet membership type. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `maxtop` | `int32` | Maximum number of top players to return. Use a large number to get entire leaderboard. |
| `modes` | `string` | List of game modes for which to get leaderboards. See the documentation for DestinyActivityModeType for valid values, and pass in string representation, comma delimited. |
| `statid` | `string` | ID of stat to return rather than returning all Leaderboard stats. |

#### Destiny2.GetLeaderboardsForCharacter

`GET /Destiny2/Stats/Leaderboards/{membershipType}/{destinyMembershipId}/{characterId}/`

Gets leaderboards with the signed in user's friends and the supplied destinyMembershipId as the focus. PREVIEW: This endpoint is still in beta, and may experience rough edges. The schema is in final form, but there may be bugs that prevent desirable operation.

- **Response:** `object`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `characterId` | `int64` | The specific character to build the leaderboard around for the provided Destiny Membership. |
| `destinyMembershipId` | `int64` | The Destiny membershipId of the user to retrieve. |
| `membershipType` | `int32` | A valid non-BungieNet membership type. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `maxtop` | `int32` | Maximum number of top players to return. Use a large number to get entire leaderboard. |
| `modes` | `string` | List of game modes for which to get leaderboards. See the documentation for DestinyActivityModeType for valid values, and pass in string representation, comma delimited. |
| `statid` | `string` | ID of stat to return rather than returning all Leaderboard stats. |

#### Destiny2.SearchDestinyEntities

`GET /Destiny2/Armory/Search/{type}/{searchTerm}/`

Gets a page list of Destiny items.

- **Response:** `Destiny.Definitions.DestinyEntitySearchResult`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `searchTerm` | `string` | The string to use when searching for Destiny entities. |
| `type` | `string` | The type of entity for whom you would like results. These correspond to the entity's definition contract name. For instance, if you are looking for items, this property should be 'DestinyInventoryItemDefinition'. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `page` | `int32` | Page number to return, starting with 0. |

#### Destiny2.GetHistoricalStats

`GET /Destiny2/{membershipType}/Account/{destinyMembershipId}/Character/{characterId}/Stats/`

Gets historical stats for indicated character.

- **Response:** `object`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `characterId` | `int64` | The id of the character to retrieve. You can omit this character ID or set it to 0 to get aggregate stats across all characters. |
| `destinyMembershipId` | `int64` | The Destiny membershipId of the user to retrieve. |
| `membershipType` | `int32` | A valid non-BungieNet membership type. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `dayend` | `date-time` | Last day to return when daily stats are requested. Use the format YYYY-MM-DD. Currently, we cannot allow more than 31 days of daily data to be requested in a single request. |
| `daystart` | `date-time` | First day to return when daily stats are requested. Use the format YYYY-MM-DD. Currently, we cannot allow more than 31 days of daily data to be requested in a single request. |
| `groups` | `array<int32>` | Group of stats to include, otherwise only general stats are returned. Comma separated list is allowed. Values: General, Weapons, Medals |
| `modes` | `array<int32>` | Game modes to return. See the documentation for DestinyActivityModeType for valid values, and pass in string representation, comma delimited. |
| `periodType` | `int32` | Indicates a specific period type to return. Optional. May be: Daily, AllTime, or Activity |

#### Destiny2.GetHistoricalStatsForAccount

`GET /Destiny2/{membershipType}/Account/{destinyMembershipId}/Stats/`

Gets aggregate historical stats organized around each character for a given account.

- **Response:** `Destiny.HistoricalStats.DestinyHistoricalStatsAccountResult`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `destinyMembershipId` | `int64` | The Destiny membershipId of the user to retrieve. |
| `membershipType` | `int32` | A valid non-BungieNet membership type. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groups` | `array<int32>` | Groups of stats to include, otherwise only general stats are returned. Comma separated list is allowed. Values: General, Weapons, Medals. |

#### Destiny2.GetActivityHistory

`GET /Destiny2/{membershipType}/Account/{destinyMembershipId}/Character/{characterId}/Stats/Activities/`

Gets activity history stats for indicated character.

- **Response:** `Destiny.HistoricalStats.DestinyActivityHistoryResults`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `characterId` | `int64` | The id of the character to retrieve. |
| `destinyMembershipId` | `int64` | The Destiny membershipId of the user to retrieve. |
| `membershipType` | `int32` | A valid non-BungieNet membership type. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `count` | `int32` | Number of rows to return |
| `mode` | `int32` | A filter for the activity mode to be returned. None returns all activities. See the documentation for DestinyActivityModeType for valid values, and pass in string representation. For historical reasons, this list will have both D1 and D2-relevant Activity Modes in it. Please don't take this to mean that some D1-only feature is coming back! |
| `page` | `int32` | Page number to return, starting with 0. |

#### Destiny2.GetUniqueWeaponHistory

`GET /Destiny2/{membershipType}/Account/{destinyMembershipId}/Character/{characterId}/Stats/UniqueWeapons/`

Gets details about unique weapon usage, including all exotic weapons.

- **Response:** `Destiny.HistoricalStats.DestinyHistoricalWeaponStatsData`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `characterId` | `int64` | The id of the character to retrieve. |
| `destinyMembershipId` | `int64` | The Destiny membershipId of the user to retrieve. |
| `membershipType` | `int32` | A valid non-BungieNet membership type. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

#### Destiny2.GetDestinyAggregateActivityStats

`GET /Destiny2/{membershipType}/Account/{destinyMembershipId}/Character/{characterId}/Stats/AggregateActivityStats/`

Gets all activities the character has participated in together with aggregate statistics for those activities.

- **Response:** `Destiny.HistoricalStats.DestinyAggregateActivityResults`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `characterId` | `int64` | The specific character whose activities should be returned. |
| `destinyMembershipId` | `int64` | The Destiny membershipId of the user to retrieve. |
| `membershipType` | `int32` | A valid non-BungieNet membership type. The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType. |

#### Destiny2.GetPublicMilestoneContent

`GET /Destiny2/Milestones/{milestoneHash}/Content/`

Gets custom localized content for the milestone of the given hash, if it exists.

- **Response:** `Destiny.Milestones.DestinyMilestoneContent`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `milestoneHash` | `uint32` | The identifier for the milestone to be returned. |

#### Destiny2.GetPublicMilestones

`GET /Destiny2/Milestones/`

Gets public information about currently available Milestones.

- **Response:** `object`

#### Destiny2.AwaInitializeRequest

`POST /Destiny2/Awa/Initialize/`

Initialize a request to perform an advanced write action.

- **OAuth scope(s):** `AdvancedWriteActions`
- **Request body:** `Destiny.Advanced.AwaPermissionRequested`
- **Response:** `Destiny.Advanced.AwaInitializeResponse`

#### Destiny2.AwaProvideAuthorizationResult

`POST /Destiny2/Awa/AwaProvideAuthorizationResult/`

Provide the result of the user interaction. Called by the Bungie Destiny App to approve or reject a request.

- **Request body:** `Destiny.Advanced.AwaUserResponse`
- **Response:** `int32`

#### Destiny2.AwaGetActionToken

`GET /Destiny2/Awa/GetActionToken/{correlationId}/`

Returns the action token if user approves the request.

- **OAuth scope(s):** `AdvancedWriteActions`
- **Response:** `Destiny.Advanced.AwaAuthorizationResult`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `correlationId` | `string` | The identifier for the advanced write action request. |

### CommunityContent

#### CommunityContent.GetCommunityContent

`GET /CommunityContent/Get/{sort}/{mediaFilter}/{page}/`

Returns community content.

- **Response:** `Forum.PostSearchResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `mediaFilter` | `int32` | The type of media to get |
| `page` | `int32` | Zero based page |
| `sort` | `byte` | The sort mode. |

### Trending

#### Trending.GetTrendingCategories

`GET /Trending/Categories/`

Returns trending items for Bungie.net, collapsed into the first page of items per category. For pagination within a category, call GetTrendingCategory.

- **Response:** `Trending.TrendingCategories`

#### Trending.GetTrendingCategory

`GET /Trending/Categories/{categoryId}/{pageNumber}/`

Returns paginated lists of trending items for a category.

- **Response:** `SearchResultOfTrendingEntry`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `categoryId` | `string` | The ID of the category for whom you want additional results. |
| `pageNumber` | `int32` | The page # of results to return. |

#### Trending.GetTrendingEntryDetail

`GET /Trending/Details/{trendingEntryType}/{identifier}/`

Returns the detailed results for a specific trending entry. Note that trending entries are uniquely identified by a combination of *both* the TrendingEntryType *and* the identifier: the identifier alone is not guaranteed to be globally unique.

- **Response:** `Trending.TrendingDetail`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `identifier` | `string` | The identifier for the entity to be returned. |
| `trendingEntryType` | `int32` | The type of entity to be returned. The known entity types that you can have returned from Trending. |

### Fireteam

#### Fireteam.GetActivePrivateClanFireteamCount

`GET /Fireteam/Clan/{groupId}/ActiveCount/`

Gets a count of all active non-public fireteams for the specified clan. Maximum value returned is 25.

- **OAuth scope(s):** `ReadGroups`
- **Response:** `int32`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | The group id of the clan. |

#### Fireteam.GetAvailableClanFireteams

`GET /Fireteam/Clan/{groupId}/Available/{platform}/{activityType}/{dateRange}/{slotFilter}/{publicOnly}/{page}/`

Gets a listing of all of this clan's fireteams that are have available slots. Caller is not checked for join criteria so caching is maximized.

- **OAuth scope(s):** `ReadGroups`
- **Response:** `SearchResultOfFireteamSummary`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `activityType` | `int32` | The activity type to filter by. |
| `dateRange` | `byte` | The date range to grab available fireteams. |
| `groupId` | `int64` | The group id of the clan. |
| `page` | `int32` | Zero based page |
| `platform` | `byte` | The platform filter. |
| `publicOnly` | `byte` | Determines public/private filtering. |
| `slotFilter` | `byte` | Filters based on available slots |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `excludeImmediate` | `boolean` | If you wish the result to exclude immediate fireteams, set this to true. Immediate-only can be forced using the dateRange enum. |
| `langFilter` | `string` | An optional language filter. |

#### Fireteam.SearchPublicAvailableClanFireteams

`GET /Fireteam/Search/Available/{platform}/{activityType}/{dateRange}/{slotFilter}/{page}/`

Gets a listing of all public fireteams starting now with open slots. Caller is not checked for join criteria so caching is maximized.

- **OAuth scope(s):** `ReadGroups`
- **Response:** `SearchResultOfFireteamSummary`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `activityType` | `int32` | The activity type to filter by. |
| `dateRange` | `byte` | The date range to grab available fireteams. |
| `page` | `int32` | Zero based page |
| `platform` | `byte` | The platform filter. |
| `slotFilter` | `byte` | Filters based on available slots |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `excludeImmediate` | `boolean` | If you wish the result to exclude immediate fireteams, set this to true. Immediate-only can be forced using the dateRange enum. |
| `langFilter` | `string` | An optional language filter. |

#### Fireteam.GetMyClanFireteams

`GET /Fireteam/Clan/{groupId}/My/{platform}/{includeClosed}/{page}/`

Gets a listing of all fireteams that caller is an applicant, a member, or an alternate of.

- **OAuth scope(s):** `ReadGroups`
- **Response:** `SearchResultOfFireteamResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupId` | `int64` | The group id of the clan. (This parameter is ignored unless the optional query parameter groupFilter is true). |
| `includeClosed` | `boolean` | If true, return fireteams that have been closed. |
| `page` | `int32` | Deprecated parameter, ignored. |
| `platform` | `byte` | The platform filter. |

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `groupFilter` | `boolean` | If true, filter by clan. Otherwise, ignore the clan and show all of the user's fireteams. |
| `langFilter` | `string` | An optional language filter. |

#### Fireteam.GetClanFireteam

`GET /Fireteam/Clan/{groupId}/Summary/{fireteamId}/`

Gets a specific fireteam.

- **OAuth scope(s):** `ReadGroups`
- **Response:** `Fireteam.FireteamResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `fireteamId` | `int64` | The unique id of the fireteam. |
| `groupId` | `int64` | The group id of the clan. |

### Social

#### Social.GetFriendList

`GET /Social/Friends/`

Returns your Bungie Friend list

- **OAuth scope(s):** `ReadUserData`
- **Response:** `Social.Friends.BungieFriendListResponse`

#### Social.GetFriendRequestList

`GET /Social/Friends/Requests/`

Returns your friend request queue.

- **OAuth scope(s):** `ReadUserData`
- **Response:** `Social.Friends.BungieFriendRequestListResponse`

#### Social.IssueFriendRequest

`POST /Social/Friends/Add/{membershipId}/`

Requests a friend relationship with the target user. Any of the target user's linked membership ids are valid inputs.

- **OAuth scope(s):** `BnetWrite`
- **Response:** `boolean`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `membershipId` | `string` | The membership id of the user you wish to add. |

#### Social.AcceptFriendRequest

`POST /Social/Friends/Requests/Accept/{membershipId}/`

Accepts a friend relationship with the target user. The user must be on your incoming friend request list, though no error will occur if they are not.

- **OAuth scope(s):** `BnetWrite`
- **Response:** `boolean`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `membershipId` | `string` | The membership id of the user you wish to accept. |

#### Social.DeclineFriendRequest

`POST /Social/Friends/Requests/Decline/{membershipId}/`

Declines a friend relationship with the target user. The user must be on your incoming friend request list, though no error will occur if they are not.

- **OAuth scope(s):** `BnetWrite`
- **Response:** `boolean`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `membershipId` | `string` | The membership id of the user you wish to decline. |

#### Social.RemoveFriend

`POST /Social/Friends/Remove/{membershipId}/`

Remove a friend relationship with the target user. The user must be on your friend list, though no error will occur if they are not.

- **OAuth scope(s):** `BnetWrite`
- **Response:** `boolean`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `membershipId` | `string` | The membership id of the user you wish to remove. |

#### Social.RemoveFriendRequest

`POST /Social/Friends/Requests/Remove/{membershipId}/`

Remove a friend relationship with the target user. The user must be on your outgoing request friend list, though no error will occur if they are not.

- **OAuth scope(s):** `BnetWrite`
- **Response:** `boolean`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `membershipId` | `string` | The membership id of the user you wish to remove. |

#### Social.GetPlatformFriendList

`GET /Social/PlatformFriends/{friendPlatform}/{page}/`

Gets the platform friend of the requested type, with additional information if they have Bungie accounts. Must have a recent login session with said platform.

- **Response:** `Social.Friends.PlatformFriendResponse`

**Path parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `friendPlatform` | `int32` | The platform friend type. |
| `page` | `string` | The zero based page to return. Page size is 100. |

### Core

#### Core.GetAvailableLocales

`GET /GetAvailableLocales/`

List of available localization cultures

- **Response:** `object`

#### Core.GetCommonSettings

`GET /Settings/`

Get the common settings used by the Bungie.Net environment.

- **Response:** `Common.Models.CoreSettingsConfiguration`

#### Core.GetUserSystemOverrides

`GET /UserSystemOverrides/`

Get the user-specific system overrides that should be respected alongside common systems.

- **Response:** `object`

#### Core.GetGlobalAlerts

`GET /GlobalAlerts/`

Gets any active global alert for display in the forum banners, help pages, etc. Usually used for DOC alerts.

- **Response:** `array<GlobalAlert>`

**Query parameters:**

| Name | Type | Description |
| --- | --- | --- |
| `includestreaming` | `boolean` | Determines whether Streaming Alerts are included in results |

## Entity index

828 entities (676 objects, 152 enums), grouped by namespace.

- **(root types)** (113): [BungieMembershipTypeEnumeration](#bungiemembershiptypeenumeration), [BungieCredentialTypeEnumeration](#bungiecredentialtypeenumeration), [SearchResultOfContentItemPublicContract](#searchresultofcontentitempubliccontract), [SearchResultOfPostResponse](#searchresultofpostresponse), [BungieMembershipType[]](#bungiemembershiptype), [SearchResultOfGroupV2Card](#searchresultofgroupv2card), [SearchResultOfGroupMember](#searchresultofgroupmember), [SearchResultOfGroupBan](#searchresultofgroupban), [SearchResultOfGroupEditHistory](#searchresultofgroupedithistory), [SearchResultOfGroupMemberApplication](#searchresultofgroupmemberapplication), [SearchResultOfGroupMembership](#searchresultofgroupmembership), [SearchResultOfGroupPotentialMembership](#searchresultofgrouppotentialmembership), [SingleComponentResponseOfDestinyVendorReceiptsComponent](#singlecomponentresponseofdestinyvendorreceiptscomponent), [SingleComponentResponseOfDestinyInventoryComponent](#singlecomponentresponseofdestinyinventorycomponent), [SingleComponentResponseOfDestinyProfileComponent](#singlecomponentresponseofdestinyprofilecomponent), [SingleComponentResponseOfDestinyPlatformSilverComponent](#singlecomponentresponseofdestinyplatformsilvercomponent), [SingleComponentResponseOfDestinyKiosksComponent](#singlecomponentresponseofdestinykioskscomponent), [SingleComponentResponseOfDestinyPlugSetsComponent](#singlecomponentresponseofdestinyplugsetscomponent), [SingleComponentResponseOfDestinyProfileProgressionComponent](#singlecomponentresponseofdestinyprofileprogressioncomponent), [SingleComponentResponseOfDestinyPresentationNodesComponent](#singlecomponentresponseofdestinypresentationnodescomponent), [SingleComponentResponseOfDestinyProfileRecordsComponent](#singlecomponentresponseofdestinyprofilerecordscomponent), [SingleComponentResponseOfDestinyProfileCollectiblesComponent](#singlecomponentresponseofdestinyprofilecollectiblescomponent), [SingleComponentResponseOfDestinyProfileTransitoryComponent](#singlecomponentresponseofdestinyprofiletransitorycomponent), [SingleComponentResponseOfDestinyMetricsComponent](#singlecomponentresponseofdestinymetricscomponent), [SingleComponentResponseOfDestinyStringVariablesComponent](#singlecomponentresponseofdestinystringvariablescomponent), [SingleComponentResponseOfDestinySocialCommendationsComponent](#singlecomponentresponseofdestinysocialcommendationscomponent), [DictionaryComponentResponseOfint64AndDestinyCharacterComponent](#dictionarycomponentresponseofint64anddestinycharactercomponent), [DictionaryComponentResponseOfint64AndDestinyInventoryComponent](#dictionarycomponentresponseofint64anddestinyinventorycomponent), [DictionaryComponentResponseOfint64AndDestinyLoadoutsComponent](#dictionarycomponentresponseofint64anddestinyloadoutscomponent), [DictionaryComponentResponseOfint64AndDestinyCharacterProgressionComponent](#dictionarycomponentresponseofint64anddestinycharacterprogressioncomponent), [DictionaryComponentResponseOfint64AndDestinyCharacterRenderComponent](#dictionarycomponentresponseofint64anddestinycharacterrendercomponent), [DictionaryComponentResponseOfint64AndDestinyCharacterActivitiesComponent](#dictionarycomponentresponseofint64anddestinycharacteractivitiescomponent), [DictionaryComponentResponseOfint64AndDestinyKiosksComponent](#dictionarycomponentresponseofint64anddestinykioskscomponent), [DictionaryComponentResponseOfint64AndDestinyPlugSetsComponent](#dictionarycomponentresponseofint64anddestinyplugsetscomponent), [DestinyBaseItemComponentSetOfuint32](#destinybaseitemcomponentsetofuint32), [DictionaryComponentResponseOfuint32AndDestinyItemObjectivesComponent](#dictionarycomponentresponseofuint32anddestinyitemobjectivescomponent), [DictionaryComponentResponseOfuint32AndDestinyItemPerksComponent](#dictionarycomponentresponseofuint32anddestinyitemperkscomponent), [DictionaryComponentResponseOfint64AndDestinyPresentationNodesComponent](#dictionarycomponentresponseofint64anddestinypresentationnodescomponent), [DictionaryComponentResponseOfint64AndDestinyCharacterRecordsComponent](#dictionarycomponentresponseofint64anddestinycharacterrecordscomponent), [DictionaryComponentResponseOfint64AndDestinyCollectiblesComponent](#dictionarycomponentresponseofint64anddestinycollectiblescomponent), [DictionaryComponentResponseOfint64AndDestinyStringVariablesComponent](#dictionarycomponentresponseofint64anddestinystringvariablescomponent), [DictionaryComponentResponseOfint64AndDestinyCraftablesComponent](#dictionarycomponentresponseofint64anddestinycraftablescomponent), [DestinyBaseItemComponentSetOfint64](#destinybaseitemcomponentsetofint64), [DictionaryComponentResponseOfint64AndDestinyItemObjectivesComponent](#dictionarycomponentresponseofint64anddestinyitemobjectivescomponent), [DictionaryComponentResponseOfint64AndDestinyItemPerksComponent](#dictionarycomponentresponseofint64anddestinyitemperkscomponent), [DestinyItemComponentSetOfint64](#destinyitemcomponentsetofint64), [DictionaryComponentResponseOfint64AndDestinyItemInstanceComponent](#dictionarycomponentresponseofint64anddestinyiteminstancecomponent), [DictionaryComponentResponseOfint64AndDestinyItemRenderComponent](#dictionarycomponentresponseofint64anddestinyitemrendercomponent), [DictionaryComponentResponseOfint64AndDestinyItemStatsComponent](#dictionarycomponentresponseofint64anddestinyitemstatscomponent), [DictionaryComponentResponseOfint64AndDestinyItemSocketsComponent](#dictionarycomponentresponseofint64anddestinyitemsocketscomponent), [DictionaryComponentResponseOfint64AndDestinyItemReusablePlugsComponent](#dictionarycomponentresponseofint64anddestinyitemreusableplugscomponent), [DictionaryComponentResponseOfint64AndDestinyItemPlugObjectivesComponent](#dictionarycomponentresponseofint64anddestinyitemplugobjectivescomponent), [DictionaryComponentResponseOfint64AndDestinyItemTalentGridComponent](#dictionarycomponentresponseofint64anddestinyitemtalentgridcomponent), [DictionaryComponentResponseOfuint32AndDestinyItemPlugComponent](#dictionarycomponentresponseofuint32anddestinyitemplugcomponent), [DictionaryComponentResponseOfint64AndDestinyCurrenciesComponent](#dictionarycomponentresponseofint64anddestinycurrenciescomponent), [SingleComponentResponseOfDestinyCharacterComponent](#singlecomponentresponseofdestinycharactercomponent), [SingleComponentResponseOfDestinyCharacterProgressionComponent](#singlecomponentresponseofdestinycharacterprogressioncomponent), [SingleComponentResponseOfDestinyCharacterRenderComponent](#singlecomponentresponseofdestinycharacterrendercomponent), [SingleComponentResponseOfDestinyCharacterActivitiesComponent](#singlecomponentresponseofdestinycharacteractivitiescomponent), [SingleComponentResponseOfDestinyLoadoutsComponent](#singlecomponentresponseofdestinyloadoutscomponent), [SingleComponentResponseOfDestinyCharacterRecordsComponent](#singlecomponentresponseofdestinycharacterrecordscomponent), [SingleComponentResponseOfDestinyCollectiblesComponent](#singlecomponentresponseofdestinycollectiblescomponent), [SingleComponentResponseOfDestinyCurrenciesComponent](#singlecomponentresponseofdestinycurrenciescomponent), [SingleComponentResponseOfDestinyItemComponent](#singlecomponentresponseofdestinyitemcomponent), [SingleComponentResponseOfDestinyItemInstanceComponent](#singlecomponentresponseofdestinyiteminstancecomponent), [SingleComponentResponseOfDestinyItemObjectivesComponent](#singlecomponentresponseofdestinyitemobjectivescomponent), [SingleComponentResponseOfDestinyItemPerksComponent](#singlecomponentresponseofdestinyitemperkscomponent), [SingleComponentResponseOfDestinyItemRenderComponent](#singlecomponentresponseofdestinyitemrendercomponent), [SingleComponentResponseOfDestinyItemStatsComponent](#singlecomponentresponseofdestinyitemstatscomponent), [SingleComponentResponseOfDestinyItemTalentGridComponent](#singlecomponentresponseofdestinyitemtalentgridcomponent), [SingleComponentResponseOfDestinyItemSocketsComponent](#singlecomponentresponseofdestinyitemsocketscomponent), [SingleComponentResponseOfDestinyItemReusablePlugsComponent](#singlecomponentresponseofdestinyitemreusableplugscomponent), [SingleComponentResponseOfDestinyItemPlugObjectivesComponent](#singlecomponentresponseofdestinyitemplugobjectivescomponent), [SingleComponentResponseOfDestinyVendorGroupComponent](#singlecomponentresponseofdestinyvendorgroupcomponent), [DictionaryComponentResponseOfuint32AndDestinyVendorComponent](#dictionarycomponentresponseofuint32anddestinyvendorcomponent), [DictionaryComponentResponseOfuint32AndDestinyVendorCategoriesComponent](#dictionarycomponentresponseofuint32anddestinyvendorcategoriescomponent), [DestinyVendorSaleItemSetComponentOfDestinyVendorSaleItemComponentDepends on Component "VendorSales"](#destinyvendorsaleitemsetcomponentofdestinyvendorsaleitemcomponentdepends-on-component-vendorsales), [DictionaryComponentResponseOfuint32AndPersonalDestinyVendorSaleItemSetComponent](#dictionarycomponentresponseofuint32andpersonaldestinyvendorsaleitemsetcomponent), [DestinyBaseItemComponentSetOfint32](#destinybaseitemcomponentsetofint32), [DictionaryComponentResponseOfint32AndDestinyItemObjectivesComponent](#dictionarycomponentresponseofint32anddestinyitemobjectivescomponent), [DictionaryComponentResponseOfint32AndDestinyItemPerksComponent](#dictionarycomponentresponseofint32anddestinyitemperkscomponent), [DestinyItemComponentSetOfint32](#destinyitemcomponentsetofint32), [DictionaryComponentResponseOfint32AndDestinyItemInstanceComponent](#dictionarycomponentresponseofint32anddestinyiteminstancecomponent), [DictionaryComponentResponseOfint32AndDestinyItemRenderComponent](#dictionarycomponentresponseofint32anddestinyitemrendercomponent), [DictionaryComponentResponseOfint32AndDestinyItemStatsComponent](#dictionarycomponentresponseofint32anddestinyitemstatscomponent), [DictionaryComponentResponseOfint32AndDestinyItemSocketsComponent](#dictionarycomponentresponseofint32anddestinyitemsocketscomponent), [DictionaryComponentResponseOfint32AndDestinyItemReusablePlugsComponent](#dictionarycomponentresponseofint32anddestinyitemreusableplugscomponent), [DictionaryComponentResponseOfint32AndDestinyItemPlugObjectivesComponent](#dictionarycomponentresponseofint32anddestinyitemplugobjectivescomponent), [DictionaryComponentResponseOfint32AndDestinyItemTalentGridComponent](#dictionarycomponentresponseofint32anddestinyitemtalentgridcomponent), [DestinyVendorItemComponentSetOfint32](#destinyvendoritemcomponentsetofint32), [DictionaryComponentResponseOfint32AndDestinyItemComponent](#dictionarycomponentresponseofint32anddestinyitemcomponent), [SingleComponentResponseOfDestinyVendorComponent](#singlecomponentresponseofdestinyvendorcomponent), [SingleComponentResponseOfDestinyVendorCategoriesComponent](#singlecomponentresponseofdestinyvendorcategoriescomponent), [DictionaryComponentResponseOfint32AndDestinyVendorSaleItemComponent](#dictionarycomponentresponseofint32anddestinyvendorsaleitemcomponent), [DictionaryComponentResponseOfuint32AndDestinyPublicVendorComponent](#dictionarycomponentresponseofuint32anddestinypublicvendorcomponent), [DestinyVendorSaleItemSetComponentOfDestinyPublicVendorSaleItemComponentDepends on Component "VendorSales"](#destinyvendorsaleitemsetcomponentofdestinypublicvendorsaleitemcomponentdepends-on-component-vendorsales), [DictionaryComponentResponseOfuint32AndPublicDestinyVendorSaleItemSetComponent](#dictionarycomponentresponseofuint32andpublicdestinyvendorsaleitemsetcomponent), [DestinyItemComponentSetOfuint32](#destinyitemcomponentsetofuint32), [DictionaryComponentResponseOfuint32AndDestinyItemInstanceComponent](#dictionarycomponentresponseofuint32anddestinyiteminstancecomponent), [DictionaryComponentResponseOfuint32AndDestinyItemRenderComponent](#dictionarycomponentresponseofuint32anddestinyitemrendercomponent), [DictionaryComponentResponseOfuint32AndDestinyItemStatsComponent](#dictionarycomponentresponseofuint32anddestinyitemstatscomponent), [DictionaryComponentResponseOfuint32AndDestinyItemSocketsComponent](#dictionarycomponentresponseofuint32anddestinyitemsocketscomponent), [DictionaryComponentResponseOfuint32AndDestinyItemReusablePlugsComponent](#dictionarycomponentresponseofuint32anddestinyitemreusableplugscomponent), [DictionaryComponentResponseOfuint32AndDestinyItemPlugObjectivesComponent](#dictionarycomponentresponseofuint32anddestinyitemplugobjectivescomponent), [DictionaryComponentResponseOfuint32AndDestinyItemTalentGridComponent](#dictionarycomponentresponseofuint32anddestinyitemtalentgridcomponent), [SearchResultOfDestinyEntitySearchResultItem](#searchresultofdestinyentitysearchresultitem), [SearchResultOfTrendingEntry](#searchresultoftrendingentry), [SearchResultOfFireteamSummary](#searchresultoffireteamsummary), [SearchResultOfFireteamResponse](#searchresultoffireteamresponse), [GlobalAlert](#globalalert), [GlobalAlertLevelEnumeration](#globalalertlevelenumeration), [GlobalAlertTypeEnumeration](#globalalerttypeenumeration), [StreamInfo](#streaminfo)
- **Applications** (9): [Applications.ApplicationScopesEnumeration](#applicationsapplicationscopesenumeration), [Applications.ApiUsage](#applicationsapiusage), [Applications.Series](#applicationsseries), [Applications.Datapoint](#applicationsdatapoint), [Applications.Application](#applicationsapplication), [Applications.OAuthApplicationTypeEnumeration](#applicationsoauthapplicationtypeenumeration), [Applications.ApplicationStatusEnumeration](#applicationsapplicationstatusenumeration), [Applications.ApplicationDeveloper](#applicationsapplicationdeveloper), [Applications.DeveloperRoleEnumeration](#applicationsdeveloperroleenumeration)
- **Common** (4): [Common.Models.CoreSettingsConfiguration](#commonmodelscoresettingsconfiguration), [Common.Models.CoreSystem](#commonmodelscoresystem), [Common.Models.CoreSetting](#commonmodelscoresetting), [Common.Models.Destiny2CoreSettings](#commonmodelsdestiny2coresettings)
- **Components** (2): [Components.ComponentResponse](#componentscomponentresponse), [Components.ComponentPrivacySettingEnumeration](#componentscomponentprivacysettingenumeration)
- **Config** (4): [Config.UserTheme](#configusertheme), [Config.GroupTheme](#configgrouptheme), [Config.ClanBanner.ClanBannerSource](#configclanbannerclanbannersource), [Config.ClanBanner.ClanBannerDecal](#configclanbannerclanbannerdecal)
- **Content** (13): [Content.Models.ContentTypeDescription](#contentmodelscontenttypedescription), [Content.Models.ContentTypeProperty](#contentmodelscontenttypeproperty), [Content.Models.ContentPropertyDataTypeEnumEnumeration](#contentmodelscontentpropertydatatypeenumenumeration), [Content.Models.ContentTypeDefaultValue](#contentmodelscontenttypedefaultvalue), [Content.Models.TagMetadataDefinition](#contentmodelstagmetadatadefinition), [Content.Models.TagMetadataItem](#contentmodelstagmetadataitem), [Content.Models.ContentPreview](#contentmodelscontentpreview), [Content.Models.ContentTypePropertySection](#contentmodelscontenttypepropertysection), [Content.ContentItemPublicContract](#contentcontentitempubliccontract), [Content.ContentRepresentation](#contentcontentrepresentation), [Content.CommentSummary](#contentcommentsummary), [Content.NewsArticleRssResponse](#contentnewsarticlerssresponse), [Content.NewsArticleRssItem](#contentnewsarticlerssitem)
- **Dates** (1): [Dates.DateRange](#datesdaterange)
- **Destiny** (547): [Destiny.DestinyProgression](#destinydestinyprogression), [Destiny.DestinyProgressionResetEntry](#destinydestinyprogressionresetentry), [Destiny.DestinyProgressionRewardItemStateEnumeration](#destinydestinyprogressionrewarditemstateenumeration), [Destiny.DestinyProgressionRewardItemSocketOverrideState](#destinydestinyprogressionrewarditemsocketoverridestate), [Destiny.DestinyStat](#destinydestinystat), [Destiny.Definitions.DestinyDefinition](#destinydefinitionsdestinydefinition), [Destiny.Definitions.DestinyStatDefinition](#destinydefinitionsdestinystatdefinition), [Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition](#destinydefinitionscommondestinydisplaypropertiesdefinition), [Destiny.Definitions.Common.DestinyIconSequenceDefinition](#destinydefinitionscommondestinyiconsequencedefinition), [Destiny.Definitions.Inventory.DestinyIconDefinition](#destinydefinitionsinventorydestinyicondefinition), [Destiny.DestinyStatAggregationTypeEnumeration](#destinydestinystataggregationtypeenumeration), [Destiny.DestinyStatCategoryEnumeration](#destinydestinystatcategoryenumeration), [Destiny.ItemStateEnumeration](#destinyitemstateenumeration), [Destiny.Definitions.DestinyProgressionDefinition](#destinydefinitionsdestinyprogressiondefinition), [Destiny.Definitions.DestinyProgressionDisplayPropertiesDefinition](#destinydefinitionsdestinyprogressiondisplaypropertiesdefinition), [Destiny.DestinyProgressionScopeEnumeration](#destinydestinyprogressionscopeenumeration), [Destiny.Definitions.DestinyProgressionStepDefinition](#destinydefinitionsdestinyprogressionstepdefinition), [Destiny.DestinyProgressionStepDisplayEffectEnumeration](#destinydestinyprogressionstepdisplayeffectenumeration), [Destiny.DestinyItemQuantity](#destinydestinyitemquantity), [Destiny.Definitions.DestinyInventoryItemDefinition](#destinydefinitionsdestinyinventoryitemdefinition), [Destiny.Definitions.DestinyItemTooltipNotification](#destinydefinitionsdestinyitemtooltipnotification), [Destiny.Misc.DestinyColor](#destinymiscdestinycolor), [Destiny.Definitions.DestinyItemActionBlockDefinition](#destinydefinitionsdestinyitemactionblockdefinition), [Destiny.Definitions.DestinyItemActionRequiredItemDefinition](#destinydefinitionsdestinyitemactionrequireditemdefinition), [Destiny.Definitions.DestinyProgressionRewardDefinition](#destinydefinitionsdestinyprogressionrewarddefinition), [Destiny.Definitions.DestinyProgressionMappingDefinition](#destinydefinitionsdestinyprogressionmappingdefinition), [Destiny.Definitions.DestinyItemCraftingBlockDefinition](#destinydefinitionsdestinyitemcraftingblockdefinition), [Destiny.Definitions.DestinyItemCraftingBlockBonusPlugDefinition](#destinydefinitionsdestinyitemcraftingblockbonusplugdefinition), [Destiny.Definitions.Sockets.DestinySocketTypeDefinition](#destinydefinitionssocketsdestinysockettypedefinition), [Destiny.Definitions.Sockets.DestinyInsertPlugActionDefinition](#destinydefinitionssocketsdestinyinsertplugactiondefinition), [Destiny.SocketTypeActionTypeEnumeration](#destinysockettypeactiontypeenumeration), [Destiny.Definitions.Sockets.DestinyPlugWhitelistEntryDefinition](#destinydefinitionssocketsdestinyplugwhitelistentrydefinition), [Destiny.DestinySocketVisibilityEnumeration](#destinydestinysocketvisibilityenumeration), [Destiny.Definitions.Sockets.DestinySocketTypeScalarMaterialRequirementEntry](#destinydefinitionssocketsdestinysockettypescalarmaterialrequiremententry), [Destiny.Definitions.Sockets.DestinySocketCategoryDefinition](#destinydefinitionssocketsdestinysocketcategorydefinition), [Destiny.DestinySocketCategoryStyleEnumeration](#destinydestinysocketcategorystyleenumeration), [Destiny.Definitions.DestinyMaterialRequirementSetDefinition](#destinydefinitionsdestinymaterialrequirementsetdefinition), [Destiny.Definitions.DestinyMaterialRequirement](#destinydefinitionsdestinymaterialrequirement), [Destiny.Definitions.DestinyItemInventoryBlockDefinition](#destinydefinitionsdestinyiteminventoryblockdefinition), [Destiny.TierTypeEnumeration](#destinytiertypeenumeration), [Destiny.Definitions.DestinyInventoryBucketDefinition](#destinydefinitionsdestinyinventorybucketdefinition), [Destiny.BucketScopeEnumeration](#destinybucketscopeenumeration), [Destiny.BucketCategoryEnumeration](#destinybucketcategoryenumeration), [Destiny.ItemLocationEnumeration](#destinyitemlocationenumeration), [Destiny.Definitions.Items.DestinyItemTierTypeDefinition](#destinydefinitionsitemsdestinyitemtiertypedefinition), [Destiny.Definitions.Items.DestinyItemTierTypeInfusionBlock](#destinydefinitionsitemsdestinyitemtiertypeinfusionblock), [Destiny.Definitions.DestinyItemSetBlockDefinition](#destinydefinitionsdestinyitemsetblockdefinition), [Destiny.Definitions.DestinyItemSetBlockEntryDefinition](#destinydefinitionsdestinyitemsetblockentrydefinition), [Destiny.Definitions.DestinyItemStatBlockDefinition](#destinydefinitionsdestinyitemstatblockdefinition), [Destiny.Definitions.DestinyInventoryItemStatDefinition](#destinydefinitionsdestinyinventoryitemstatdefinition), [Destiny.Definitions.DestinyStatGroupDefinition](#destinydefinitionsdestinystatgroupdefinition), [Destiny.Definitions.DestinyStatDisplayDefinition](#destinydefinitionsdestinystatdisplaydefinition), [Destiny.Definitions.DestinyStatOverrideDefinition](#destinydefinitionsdestinystatoverridedefinition), [Destiny.Definitions.DestinyEquippingBlockDefinition](#destinydefinitionsdestinyequippingblockdefinition), [Destiny.EquippingItemBlockAttributesEnumeration](#destinyequippingitemblockattributesenumeration), [Destiny.DestinyAmmunitionTypeEnumeration](#destinydestinyammunitiontypeenumeration), [Destiny.Definitions.DestinyEquipmentSlotDefinition](#destinydefinitionsdestinyequipmentslotdefinition), [Destiny.Definitions.DestinyArtDyeReference](#destinydefinitionsdestinyartdyereference), [Destiny.Definitions.Items.DestinyEquipableItemSetDefinition](#destinydefinitionsitemsdestinyequipableitemsetdefinition), [Destiny.Definitions.Items.DestinyItemSetPerkDefinition](#destinydefinitionsitemsdestinyitemsetperkdefinition), [Destiny.Definitions.DestinySandboxPerkDefinition](#destinydefinitionsdestinysandboxperkdefinition), [Destiny.DamageTypeEnumeration](#destinydamagetypeenumeration), [Destiny.Definitions.DestinyDamageTypeDefinition](#destinydefinitionsdestinydamagetypedefinition), [Destiny.Definitions.DestinyItemTranslationBlockDefinition](#destinydefinitionsdestinyitemtranslationblockdefinition), [Destiny.DyeReference](#destinydyereference), [Destiny.Definitions.DestinyGearArtArrangementReference](#destinydefinitionsdestinygearartarrangementreference), [Destiny.Definitions.DestinyClassDefinition](#destinydefinitionsdestinyclassdefinition), [Destiny.DestinyClassEnumeration](#destinydestinyclassenumeration), [Destiny.DestinyGenderEnumeration](#destinydestinygenderenumeration), [Destiny.Definitions.DestinyGenderDefinition](#destinydefinitionsdestinygenderdefinition), [Destiny.Definitions.DestinyVendorDefinition](#destinydefinitionsdestinyvendordefinition), [Destiny.Definitions.DestinyVendorDisplayPropertiesDefinition](#destinydefinitionsdestinyvendordisplaypropertiesdefinition), [Destiny.Definitions.DestinyVendorRequirementDisplayEntryDefinition](#destinydefinitionsdestinyvendorrequirementdisplayentrydefinition), [Destiny.DestinyVendorProgressionTypeEnumeration](#destinydestinyvendorprogressiontypeenumeration), [Destiny.Definitions.DestinyVendorActionDefinition](#destinydefinitionsdestinyvendoractiondefinition), [Destiny.Definitions.DestinyVendorCategoryEntryDefinition](#destinydefinitionsdestinyvendorcategoryentrydefinition), [Destiny.Definitions.DestinyVendorCategoryOverlayDefinition](#destinydefinitionsdestinyvendorcategoryoverlaydefinition), [Destiny.Definitions.DestinyDisplayCategoryDefinition](#destinydefinitionsdestinydisplaycategorydefinition), [Destiny.VendorDisplayCategorySortOrderEnumeration](#destinyvendordisplaycategorysortorderenumeration), [Destiny.Definitions.DestinyVendorInteractionDefinition](#destinydefinitionsdestinyvendorinteractiondefinition), [Destiny.Definitions.DestinyVendorInteractionReplyDefinition](#destinydefinitionsdestinyvendorinteractionreplydefinition), [Destiny.DestinyVendorInteractionRewardSelectionEnumeration](#destinydestinyvendorinteractionrewardselectionenumeration), [Destiny.DestinyVendorReplyTypeEnumeration](#destinydestinyvendorreplytypeenumeration), [Destiny.Definitions.DestinyVendorInteractionSackEntryDefinition](#destinydefinitionsdestinyvendorinteractionsackentrydefinition), [Destiny.VendorInteractionTypeEnumeration](#destinyvendorinteractiontypeenumeration), [Destiny.Definitions.DestinyVendorInventoryFlyoutDefinition](#destinydefinitionsdestinyvendorinventoryflyoutdefinition), [Destiny.Definitions.DestinyVendorInventoryFlyoutBucketDefinition](#destinydefinitionsdestinyvendorinventoryflyoutbucketdefinition), [Destiny.DestinyItemSortTypeEnumeration](#destinydestinyitemsorttypeenumeration), [Destiny.Definitions.DestinyVendorItemDefinition](#destinydefinitionsdestinyvendoritemdefinition), [Destiny.Definitions.DestinyVendorItemQuantity](#destinydefinitionsdestinyvendoritemquantity), [Destiny.DestinyVendorItemRefundPolicyEnumeration](#destinydestinyvendoritemrefundpolicyenumeration), [Destiny.Definitions.DestinyItemCreationEntryLevelDefinition](#destinydefinitionsdestinyitemcreationentryleveldefinition), [Destiny.Definitions.DestinyVendorSaleItemActionBlockDefinition](#destinydefinitionsdestinyvendorsaleitemactionblockdefinition), [Destiny.DestinyGatingScopeEnumeration](#destinydestinygatingscopeenumeration), [Destiny.Definitions.DestinyVendorItemSocketOverride](#destinydefinitionsdestinyvendoritemsocketoverride), [Destiny.Definitions.DestinyVendorServiceDefinition](#destinydefinitionsdestinyvendorservicedefinition), [Destiny.Definitions.DestinyVendorAcceptedItemDefinition](#destinydefinitionsdestinyvendoraccepteditemdefinition), [Destiny.Definitions.Vendors.DestinyVendorLocationDefinition](#destinydefinitionsvendorsdestinyvendorlocationdefinition), [Destiny.Definitions.DestinyDestinationDefinition](#destinydefinitionsdestinydestinationdefinition), [Destiny.Definitions.DestinyActivityGraphListEntryDefinition](#destinydefinitionsdestinyactivitygraphlistentrydefinition), [Destiny.Definitions.Director.DestinyActivityGraphDefinition](#destinydefinitionsdirectordestinyactivitygraphdefinition), [Destiny.Definitions.Director.DestinyActivityGraphNodeDefinition](#destinydefinitionsdirectordestinyactivitygraphnodedefinition), [Destiny.Definitions.Common.DestinyPositionDefinition](#destinydefinitionscommondestinypositiondefinition), [Destiny.Definitions.Director.DestinyActivityGraphNodeFeaturingStateDefinition](#destinydefinitionsdirectordestinyactivitygraphnodefeaturingstatedefinition), [Destiny.ActivityGraphNodeHighlightTypeEnumeration](#destinyactivitygraphnodehighlighttypeenumeration), [Destiny.Definitions.Director.DestinyActivityGraphNodeActivityDefinition](#destinydefinitionsdirectordestinyactivitygraphnodeactivitydefinition), [Destiny.Definitions.DestinyActivityDefinition](#destinydefinitionsdestinyactivitydefinition), [Destiny.Definitions.DestinyActivityRewardDefinition](#destinydefinitionsdestinyactivityrewarddefinition), [Destiny.Definitions.DestinyActivityModifierReferenceDefinition](#destinydefinitionsdestinyactivitymodifierreferencedefinition), [Destiny.Definitions.ActivityModifiers.DestinyActivityModifierDefinition](#destinydefinitionsactivitymodifiersdestinyactivitymodifierdefinition), [Destiny.Definitions.DestinyActivityChallengeDefinition](#destinydefinitionsdestinyactivitychallengedefinition), [Destiny.Definitions.DestinyObjectiveDefinition](#destinydefinitionsdestinyobjectivedefinition), [Destiny.DestinyUnlockValueUIStyleEnumeration](#destinydestinyunlockvalueuistyleenumeration), [Destiny.Definitions.DestinyObjectivePerkEntryDefinition](#destinydefinitionsdestinyobjectiveperkentrydefinition), [Destiny.DestinyObjectiveGrantStyleEnumeration](#destinydestinyobjectivegrantstyleenumeration), [Destiny.Definitions.DestinyObjectiveStatEntryDefinition](#destinydefinitionsdestinyobjectivestatentrydefinition), [Destiny.Definitions.DestinyItemInvestmentStatDefinition](#destinydefinitionsdestinyiteminvestmentstatdefinition), [Destiny.DestinyObjectiveUiStyleEnumeration](#destinydestinyobjectiveuistyleenumeration), [Destiny.Definitions.DestinyLocationDefinition](#destinydefinitionsdestinylocationdefinition), [Destiny.Definitions.DestinyLocationReleaseDefinition](#destinydefinitionsdestinylocationreleasedefinition), [Destiny.DestinyActivityNavPointTypeEnumeration](#destinydestinyactivitynavpointtypeenumeration), [Destiny.Definitions.DestinyActivityUnlockStringDefinition](#destinydefinitionsdestinyactivityunlockstringdefinition), [Destiny.Definitions.DestinyActivityRequirementsBlock](#destinydefinitionsdestinyactivityrequirementsblock), [Destiny.Definitions.DestinyActivityRequirementLabel](#destinydefinitionsdestinyactivityrequirementlabel), [Destiny.Definitions.DestinyActivitySelectableSkullCollections](#destinydefinitionsdestinyactivityselectableskullcollections), [Destiny.Definitions.DestinyActivityPlaylistItemDefinition](#destinydefinitionsdestinyactivityplaylistitemdefinition), [Destiny.HistoricalStats.Definitions.DestinyActivityModeTypeEnumeration](#destinyhistoricalstatsdefinitionsdestinyactivitymodetypeenumeration), [Destiny.Definitions.DestinyActivityModeDefinition](#destinydefinitionsdestinyactivitymodedefinition), [Destiny.DestinyActivityModeCategoryEnumeration](#destinydestinyactivitymodecategoryenumeration), [Destiny.Definitions.DestinyActivityMatchmakingBlockDefinition](#destinydefinitionsdestinyactivitymatchmakingblockdefinition), [Destiny.Definitions.DestinyActivityGuidedBlockDefinition](#destinydefinitionsdestinyactivityguidedblockdefinition), [Destiny.Definitions.DestinyActivityLoadoutRequirementSet](#destinydefinitionsdestinyactivityloadoutrequirementset), [Destiny.Definitions.DestinyActivityLoadoutRequirement](#destinydefinitionsdestinyactivityloadoutrequirement), [Destiny.DestinyItemSubTypeEnumeration](#destinydestinyitemsubtypeenumeration), [Destiny.Definitions.DestinyActivityInsertionPointDefinition](#destinydefinitionsdestinyactivityinsertionpointdefinition), [Destiny.Constants.DestinyEnvironmentLocationMapping](#destinyconstantsdestinyenvironmentlocationmapping), [Destiny.Definitions.DestinyActivityCuratorBlockDefinition](#destinydefinitionsdestinyactivitycuratorblockdefinition), [Destiny.Definitions.DestinyActivityDurationEstimate](#destinydefinitionsdestinyactivitydurationestimate), [Destiny.Definitions.DestinyPlaceDefinition](#destinydefinitionsdestinyplacedefinition), [Destiny.Definitions.DestinyActivityTypeDefinition](#destinydefinitionsdestinyactivitytypedefinition), [Destiny.Definitions.Activities.DestinyActivityFamilyDefinition](#destinydefinitionsactivitiesdestinyactivityfamilydefinition), [Destiny.Definitions.Traits.DestinyTraitDefinition](#destinydefinitionstraitsdestinytraitdefinition), [Destiny.Definitions.Activities.DestinyActivitySkullCategoryDefinition](#destinydefinitionsactivitiesdestinyactivityskullcategorydefinition), [Destiny.Definitions.Activities.DestinyActivitySkullSubcategoryDefinition](#destinydefinitionsactivitiesdestinyactivityskullsubcategorydefinition), [Destiny.Definitions.Activities.DestinyActivityDifficultyTierCollectionDefinition](#destinydefinitionsactivitiesdestinyactivitydifficultytiercollectiondefinition), [Destiny.Definitions.Activities.DestinyActivityDifficultyTierDefinition](#destinydefinitionsactivitiesdestinyactivitydifficultytierdefinition), [Destiny.Definitions.Activities.DestinyActivitySkull](#destinydefinitionsactivitiesdestinyactivityskull), [Destiny.Definitions.Activities.DestinyActivitySkullOption](#destinydefinitionsactivitiesdestinyactivityskulloption), [Destiny.DestinyActivityDifficultyIdEnumeration](#destinydestinyactivitydifficultyidenumeration), [Destiny.DestinyActivitySkullDynamicUseEnumeration](#destinydestinyactivityskulldynamicuseenumeration), [Destiny.DestinyActivityModifierDisplayCategoryEnumeration](#destinydestinyactivitymodifierdisplaycategoryenumeration), [Destiny.DestinyActivityModifierConnotationEnumeration](#destinydestinyactivitymodifierconnotationenumeration), [Destiny.Definitions.Activities.DestinyActivitySelectableSkullExclusionGroupDefinition](#destinydefinitionsactivitiesdestinyactivityselectableskullexclusiongroupdefinition), [Destiny.DestinyActivityDifficultyTierTypeEnumeration](#destinydestinyactivitydifficultytiertypeenumeration), [Destiny.Definitions.Activities.DestinyActivityDifficultyTierSubcategoryOverride](#destinydefinitionsactivitiesdestinyactivitydifficultytiersubcategoryoverride), [Destiny.Definitions.Activities.DestinyActivitySelectableSkullCollectionDefinition](#destinydefinitionsactivitiesdestinyactivityselectableskullcollectiondefinition), [Destiny.Definitions.Activities.DestinyActivitySelectableSkullCollectionSelectionType](#destinydefinitionsactivitiesdestinyactivityselectableskullcollectionselectiontype), [Destiny.Definitions.Activities.DestinyActivitySelectableSkull](#destinydefinitionsactivitiesdestinyactivityselectableskull), [Destiny.Definitions.Activities.DestinyActivityLoadoutRestrictionDefinition](#destinydefinitionsactivitiesdestinyactivityloadoutrestrictiondefinition), [Destiny.Definitions.Director.DestinyActivityGraphNodeStateEntry](#destinydefinitionsdirectordestinyactivitygraphnodestateentry), [Destiny.DestinyGraphNodeStateEnumeration](#destinydestinygraphnodestateenumeration), [Destiny.Definitions.Director.DestinyActivityGraphArtElementDefinition](#destinydefinitionsdirectordestinyactivitygraphartelementdefinition), [Destiny.Definitions.Director.DestinyActivityGraphConnectionDefinition](#destinydefinitionsdirectordestinyactivitygraphconnectiondefinition), [Destiny.Definitions.Director.DestinyActivityGraphDisplayObjectiveDefinition](#destinydefinitionsdirectordestinyactivitygraphdisplayobjectivedefinition), [Destiny.Definitions.Director.DestinyActivityGraphDisplayProgressionDefinition](#destinydefinitionsdirectordestinyactivitygraphdisplayprogressiondefinition), [Destiny.Definitions.Director.DestinyLinkedGraphDefinition](#destinydefinitionsdirectordestinylinkedgraphdefinition), [Destiny.Definitions.Director.DestinyLinkedGraphEntryDefinition](#destinydefinitionsdirectordestinylinkedgraphentrydefinition), [Destiny.Definitions.DestinyDestinationBubbleSettingDefinition](#destinydefinitionsdestinydestinationbubblesettingdefinition), [Destiny.Definitions.DestinyBubbleDefinition](#destinydefinitionsdestinybubbledefinition), [Destiny.Definitions.DestinyVendorGroupReference](#destinydefinitionsdestinyvendorgroupreference), [Destiny.Definitions.DestinyVendorGroupDefinition](#destinydefinitionsdestinyvendorgroupdefinition), [Destiny.Definitions.DestinyFactionDefinition](#destinydefinitionsdestinyfactiondefinition), [Destiny.Definitions.DestinyFactionVendorDefinition](#destinydefinitionsdestinyfactionvendordefinition), [Destiny.Definitions.DestinySandboxPatternDefinition](#destinydefinitionsdestinysandboxpatterndefinition), [Destiny.Definitions.DestinyArrangementRegionFilterDefinition](#destinydefinitionsdestinyarrangementregionfilterdefinition), [Destiny.Definitions.DestinyItemPreviewBlockDefinition](#destinydefinitionsdestinyitempreviewblockdefinition), [Destiny.Definitions.Items.DestinyDerivedItemCategoryDefinition](#destinydefinitionsitemsdestinyderiveditemcategorydefinition), [Destiny.Definitions.Items.DestinyDerivedItemDefinition](#destinydefinitionsitemsdestinyderiveditemdefinition), [Destiny.Definitions.Artifacts.DestinyArtifactDefinition](#destinydefinitionsartifactsdestinyartifactdefinition), [Destiny.Definitions.Artifacts.DestinyArtifactTierDefinition](#destinydefinitionsartifactsdestinyartifacttierdefinition), [Destiny.Definitions.Artifacts.DestinyArtifactTierItemDefinition](#destinydefinitionsartifactsdestinyartifacttieritemdefinition), [Destiny.Definitions.DestinyItemQualityBlockDefinition](#destinydefinitionsdestinyitemqualityblockdefinition), [Destiny.Definitions.DestinyItemVersionDefinition](#destinydefinitionsdestinyitemversiondefinition), [Destiny.Definitions.PowerCaps.DestinyPowerCapDefinition](#destinydefinitionspowercapsdestinypowercapdefinition), [Destiny.Definitions.Progression.DestinyProgressionLevelRequirementDefinition](#destinydefinitionsprogressiondestinyprogressionlevelrequirementdefinition), [Destiny.Definitions.DestinyItemValueBlockDefinition](#destinydefinitionsdestinyitemvalueblockdefinition), [Destiny.Definitions.DestinyItemSourceBlockDefinition](#destinydefinitionsdestinyitemsourceblockdefinition), [Destiny.Definitions.Sources.DestinyItemSourceDefinition](#destinydefinitionssourcesdestinyitemsourcedefinition), [Destiny.Definitions.DestinyRewardSourceDefinition](#destinydefinitionsdestinyrewardsourcedefinition), [Destiny.Definitions.DestinyRewardSourceCategoryEnumeration](#destinydefinitionsdestinyrewardsourcecategoryenumeration), [Destiny.Definitions.DestinyItemVendorSourceReference](#destinydefinitionsdestinyitemvendorsourcereference), [Destiny.Definitions.DestinyItemObjectiveBlockDefinition](#destinydefinitionsdestinyitemobjectiveblockdefinition), [Destiny.Definitions.DestinyObjectiveDisplayProperties](#destinydefinitionsdestinyobjectivedisplayproperties), [Destiny.Definitions.DestinyItemMetricBlockDefinition](#destinydefinitionsdestinyitemmetricblockdefinition), [Destiny.Definitions.Presentation.DestinyPresentationNodeBaseDefinition](#destinydefinitionspresentationdestinypresentationnodebasedefinition), [Destiny.DestinyPresentationNodeTypeEnumeration](#destinydestinypresentationnodetypeenumeration), [Destiny.Definitions.Presentation.DestinyScoredPresentationNodeBaseDefinition](#destinydefinitionspresentationdestinyscoredpresentationnodebasedefinition), [Destiny.Definitions.Presentation.DestinyPresentationNodeDefinition](#destinydefinitionspresentationdestinypresentationnodedefinition), [Destiny.DestinyScopeEnumeration](#destinydestinyscopeenumeration), [Destiny.Definitions.Presentation.DestinyPresentationNodeChildrenBlock](#destinydefinitionspresentationdestinypresentationnodechildrenblock), [Destiny.Definitions.Presentation.DestinyPresentationNodeChildEntryBase](#destinydefinitionspresentationdestinypresentationnodechildentrybase), [Destiny.Definitions.Presentation.DestinyPresentationNodeChildEntry](#destinydefinitionspresentationdestinypresentationnodechildentry), [Destiny.Definitions.Presentation.DestinyPresentationNodeCollectibleChildEntry](#destinydefinitionspresentationdestinypresentationnodecollectiblechildentry), [Destiny.Definitions.Collectibles.DestinyCollectibleDefinition](#destinydefinitionscollectiblesdestinycollectibledefinition), [Destiny.Definitions.Collectibles.DestinyCollectibleAcquisitionBlock](#destinydefinitionscollectiblesdestinycollectibleacquisitionblock), [Destiny.Definitions.DestinyUnlockValueDefinition](#destinydefinitionsdestinyunlockvaluedefinition), [Destiny.Definitions.Collectibles.DestinyCollectibleStateBlock](#destinydefinitionscollectiblesdestinycollectiblestateblock), [Destiny.Definitions.Presentation.DestinyPresentationNodeRequirementsBlock](#destinydefinitionspresentationdestinypresentationnoderequirementsblock), [Destiny.Definitions.Presentation.DestinyPresentationChildBlock](#destinydefinitionspresentationdestinypresentationchildblock), [Destiny.DestinyPresentationDisplayStyleEnumeration](#destinydestinypresentationdisplaystyleenumeration), [Destiny.Definitions.Presentation.DestinyPresentationNodeRecordChildEntry](#destinydefinitionspresentationdestinypresentationnoderecordchildentry), [Destiny.Definitions.Records.DestinyRecordDefinition](#destinydefinitionsrecordsdestinyrecorddefinition), [Destiny.DestinyRecordValueStyleEnumeration](#destinydestinyrecordvaluestyleenumeration), [Destiny.Definitions.Records.DestinyRecordTitleBlock](#destinydefinitionsrecordsdestinyrecordtitleblock), [Destiny.Definitions.Records.DestinyRecordCompletionBlock](#destinydefinitionsrecordsdestinyrecordcompletionblock), [Destiny.DestinyRecordToastStyleEnumeration](#destinydestinyrecordtoaststyleenumeration), [Destiny.Definitions.Records.SchemaRecordStateBlock](#destinydefinitionsrecordsschemarecordstateblock), [Destiny.Definitions.Records.DestinyRecordExpirationBlock](#destinydefinitionsrecordsdestinyrecordexpirationblock), [Destiny.Definitions.Records.DestinyRecordIntervalBlock](#destinydefinitionsrecordsdestinyrecordintervalblock), [Destiny.Definitions.Records.DestinyRecordIntervalObjective](#destinydefinitionsrecordsdestinyrecordintervalobjective), [Destiny.Definitions.Records.DestinyRecordIntervalRewards](#destinydefinitionsrecordsdestinyrecordintervalrewards), [Destiny.Definitions.Lore.DestinyLoreDefinition](#destinydefinitionsloredestinyloredefinition), [Destiny.Definitions.Presentation.DestinyPresentationNodeMetricChildEntry](#destinydefinitionspresentationdestinypresentationnodemetricchildentry), [Destiny.Definitions.Metrics.DestinyMetricDefinition](#destinydefinitionsmetricsdestinymetricdefinition), [Destiny.Definitions.Presentation.DestinyPresentationNodeCraftableChildEntry](#destinydefinitionspresentationdestinypresentationnodecraftablechildentry), [Destiny.DestinyPresentationScreenStyleEnumeration](#destinydestinypresentationscreenstyleenumeration), [Destiny.Definitions.Items.DestinyItemPlugDefinition](#destinydefinitionsitemsdestinyitemplugdefinition), [Destiny.Definitions.Items.DestinyPlugRuleDefinition](#destinydefinitionsitemsdestinyplugruledefinition), [Destiny.PlugUiStylesEnumeration](#destinypluguistylesenumeration), [Destiny.PlugAvailabilityModeEnumeration](#destinyplugavailabilitymodeenumeration), [Destiny.Definitions.Items.DestinyParentItemOverride](#destinydefinitionsitemsdestinyparentitemoverride), [Destiny.Definitions.Items.DestinyEnergyCapacityEntry](#destinydefinitionsitemsdestinyenergycapacityentry), [Destiny.DestinyEnergyTypeEnumeration](#destinydestinyenergytypeenumeration), [Destiny.Definitions.EnergyTypes.DestinyEnergyTypeDefinition](#destinydefinitionsenergytypesdestinyenergytypedefinition), [Destiny.Definitions.Items.DestinyEnergyCostEntry](#destinydefinitionsitemsdestinyenergycostentry), [Destiny.Definitions.DestinyItemGearsetBlockDefinition](#destinydefinitionsdestinyitemgearsetblockdefinition), [Destiny.Definitions.DestinyItemSackBlockDefinition](#destinydefinitionsdestinyitemsackblockdefinition), [Destiny.Definitions.DestinyItemSocketBlockDefinition](#destinydefinitionsdestinyitemsocketblockdefinition), [Destiny.Definitions.DestinyItemSocketEntryDefinition](#destinydefinitionsdestinyitemsocketentrydefinition), [Destiny.Definitions.DestinyItemSocketEntryPlugItemDefinition](#destinydefinitionsdestinyitemsocketentryplugitemdefinition), [Destiny.SocketPlugSourcesEnumeration](#destinysocketplugsourcesenumeration), [Destiny.Definitions.Sockets.DestinyPlugSetDefinition](#destinydefinitionssocketsdestinyplugsetdefinition), [Destiny.Definitions.DestinyItemSocketEntryPlugItemRandomizedDefinition](#destinydefinitionsdestinyitemsocketentryplugitemrandomizeddefinition), [Destiny.Definitions.DestinyPlugItemCraftingRequirements](#destinydefinitionsdestinyplugitemcraftingrequirements), [Destiny.Definitions.DestinyPlugItemCraftingUnlockRequirement](#destinydefinitionsdestinyplugitemcraftingunlockrequirement), [Destiny.Definitions.DestinyItemIntrinsicSocketEntryDefinition](#destinydefinitionsdestinyitemintrinsicsocketentrydefinition), [Destiny.Definitions.DestinyItemSocketCategoryDefinition](#destinydefinitionsdestinyitemsocketcategorydefinition), [Destiny.Definitions.DestinyItemSummaryBlockDefinition](#destinydefinitionsdestinyitemsummaryblockdefinition), [Destiny.Definitions.DestinyItemTalentGridBlockDefinition](#destinydefinitionsdestinyitemtalentgridblockdefinition), [Destiny.Definitions.DestinyTalentGridDefinition](#destinydefinitionsdestinytalentgriddefinition), [Destiny.Definitions.DestinyTalentNodeDefinition](#destinydefinitionsdestinytalentnodedefinition), [Destiny.Definitions.DestinyNodeActivationRequirement](#destinydefinitionsdestinynodeactivationrequirement), [Destiny.Definitions.DestinyNodeStepDefinition](#destinydefinitionsdestinynodestepdefinition), [Destiny.Definitions.DestinyTalentNodeStepGroups](#destinydefinitionsdestinytalentnodestepgroups), [Destiny.Definitions.DestinyTalentNodeStepWeaponPerformancesEnumeration](#destinydefinitionsdestinytalentnodestepweaponperformancesenumeration), [Destiny.Definitions.DestinyTalentNodeStepImpactEffectsEnumeration](#destinydefinitionsdestinytalentnodestepimpacteffectsenumeration), [Destiny.Definitions.DestinyTalentNodeStepGuardianAttributesEnumeration](#destinydefinitionsdestinytalentnodestepguardianattributesenumeration), [Destiny.Definitions.DestinyTalentNodeStepLightAbilitiesEnumeration](#destinydefinitionsdestinytalentnodesteplightabilitiesenumeration), [Destiny.Definitions.DestinyTalentNodeStepDamageTypesEnumeration](#destinydefinitionsdestinytalentnodestepdamagetypesenumeration), [Destiny.Definitions.DestinyNodeSocketReplaceResponse](#destinydefinitionsdestinynodesocketreplaceresponse), [Destiny.Definitions.DestinyTalentNodeExclusiveSetDefinition](#destinydefinitionsdestinytalentnodeexclusivesetdefinition), [Destiny.Definitions.DestinyTalentExclusiveGroup](#destinydefinitionsdestinytalentexclusivegroup), [Destiny.Definitions.DestinyTalentNodeCategory](#destinydefinitionsdestinytalentnodecategory), [Destiny.Definitions.DestinyItemPerkEntryDefinition](#destinydefinitionsdestinyitemperkentrydefinition), [Destiny.ItemPerkVisibilityEnumeration](#destinyitemperkvisibilityenumeration), [Destiny.Definitions.Animations.DestinyAnimationReference](#destinydefinitionsanimationsdestinyanimationreference), [Destiny.SpecialItemTypeEnumeration](#destinyspecialitemtypeenumeration), [Destiny.DestinyItemTypeEnumeration](#destinydestinyitemtypeenumeration), [Destiny.DestinyBreakerTypeEnumeration](#destinydestinybreakertypeenumeration), [Destiny.Definitions.DestinyItemCategoryDefinition](#destinydefinitionsdestinyitemcategorydefinition), [Destiny.Definitions.BreakerTypes.DestinyBreakerTypeDefinition](#destinydefinitionsbreakertypesdestinybreakertypedefinition), [Destiny.Definitions.Seasons.DestinySeasonDefinition](#destinydefinitionsseasonsdestinyseasondefinition), [Destiny.Definitions.Seasons.DestinySeasonPassReference](#destinydefinitionsseasonsdestinyseasonpassreference), [Destiny.Definitions.Seasons.DestinySeasonPassDefinition](#destinydefinitionsseasonsdestinyseasonpassdefinition), [Destiny.Definitions.Seasons.DestinySeasonPassImages](#destinydefinitionsseasonsdestinyseasonpassimages), [Destiny.Definitions.Seasons.DestinySeasonActDefinition](#destinydefinitionsseasonsdestinyseasonactdefinition), [Destiny.Definitions.Seasons.DestinySeasonPreviewDefinition](#destinydefinitionsseasonsdestinyseasonpreviewdefinition), [Destiny.Definitions.Seasons.DestinySeasonPreviewImageDefinition](#destinydefinitionsseasonsdestinyseasonpreviewimagedefinition), [Destiny.Definitions.DestinyProgressionRewardItemQuantity](#destinydefinitionsdestinyprogressionrewarditemquantity), [Destiny.DestinyProgressionRewardItemAcquisitionBehaviorEnumeration](#destinydestinyprogressionrewarditemacquisitionbehaviorenumeration), [Destiny.Definitions.DestinyProgressionSocketPlugOverride](#destinydefinitionsdestinyprogressionsocketplugoverride), [Destiny.Config.DestinyManifest](#destinyconfigdestinymanifest), [Destiny.Config.GearAssetDataBaseDefinition](#destinyconfiggearassetdatabasedefinition), [Destiny.Config.ImagePyramidEntry](#destinyconfigimagepyramidentry), [Destiny.Responses.DestinyLinkedProfilesResponse](#destinyresponsesdestinylinkedprofilesresponse), [Destiny.Responses.DestinyProfileUserInfoCard](#destinyresponsesdestinyprofileuserinfocard), [Destiny.Components.Inventory.DestinyPlatformSilverComponentDepends on Component "PlatformSilver"](#destinycomponentsinventorydestinyplatformsilvercomponentdepends-on-component-platformsilver), [Destiny.Entities.Items.DestinyItemComponent](#destinyentitiesitemsdestinyitemcomponent), [Destiny.ItemBindStatusEnumeration](#destinyitembindstatusenumeration), [Destiny.TransferStatusesEnumeration](#destinytransferstatusesenumeration), [Destiny.Quests.DestinyObjectiveProgress](#destinyquestsdestinyobjectiveprogress), [Destiny.DestinyGameVersionsEnumeration](#destinydestinygameversionsenumeration), [Destiny.Responses.DestinyErrorProfile](#destinyresponsesdestinyerrorprofile), [Destiny.DestinyComponentTypeEnumeration](#destinydestinycomponenttypeenumeration), [Destiny.Responses.DestinyProfileResponse](#destinyresponsesdestinyprofileresponse), [Destiny.Entities.Profiles.DestinyVendorReceiptsComponentDepends on Component "VendorReceipts"](#destinyentitiesprofilesdestinyvendorreceiptscomponentdepends-on-component-vendorreceipts), [Destiny.Vendors.DestinyVendorReceipt](#destinyvendorsdestinyvendorreceipt), [Destiny.Entities.Inventory.DestinyInventoryComponent](#destinyentitiesinventorydestinyinventorycomponent), [Destiny.Entities.Profiles.DestinyProfileComponentDepends on Component "Profiles"](#destinyentitiesprofilesdestinyprofilecomponentdepends-on-component-profiles), [Destiny.Definitions.Seasons.DestinyEventCardDefinition](#destinydefinitionsseasonsdestinyeventcarddefinition), [Destiny.Definitions.Seasons.DestinyEventCardImages](#destinydefinitionsseasonsdestinyeventcardimages), [Destiny.Definitions.GuardianRanks.DestinyGuardianRankDefinition](#destinydefinitionsguardianranksdestinyguardianrankdefinition), [Destiny.Components.Kiosks.DestinyKiosksComponentDepends on Component "Kiosks"](#destinycomponentskiosksdestinykioskscomponentdepends-on-component-kiosks), [Destiny.Components.Kiosks.DestinyKioskItem](#destinycomponentskiosksdestinykioskitem), [Destiny.Components.PlugSets.DestinyPlugSetsComponentDepends on Component "ItemSockets"](#destinycomponentsplugsetsdestinyplugsetscomponentdepends-on-component-itemsockets), [Destiny.Sockets.DestinyItemPlugBase](#destinysocketsdestinyitemplugbase), [Destiny.Sockets.DestinyItemPlug](#destinysocketsdestinyitemplug), [Destiny.Components.Profiles.DestinyProfileProgressionComponentDepends on Component "ProfileProgression"](#destinycomponentsprofilesdestinyprofileprogressioncomponentdepends-on-component-profileprogression), [Destiny.Artifacts.DestinyArtifactProfileScoped](#destinyartifactsdestinyartifactprofilescoped), [Destiny.Definitions.Checklists.DestinyChecklistDefinition](#destinydefinitionschecklistsdestinychecklistdefinition), [Destiny.Definitions.Checklists.DestinyChecklistEntryDefinition](#destinydefinitionschecklistsdestinychecklistentrydefinition), [Destiny.Components.Presentation.DestinyPresentationNodesComponentDepends on Component "PresentationNodes"](#destinycomponentspresentationdestinypresentationnodescomponentdepends-on-component-presentationnodes), [Destiny.Components.Presentation.DestinyPresentationNodeComponent](#destinycomponentspresentationdestinypresentationnodecomponent), [Destiny.DestinyPresentationNodeStateEnumeration](#destinydestinypresentationnodestateenumeration), [Destiny.Components.Records.DestinyRecordsComponentDepends on Component "Records"](#destinycomponentsrecordsdestinyrecordscomponentdepends-on-component-records), [Destiny.Components.Records.DestinyRecordComponent](#destinycomponentsrecordsdestinyrecordcomponent), [Destiny.DestinyRecordStateEnumeration](#destinydestinyrecordstateenumeration), [Destiny.Components.Records.DestinyProfileRecordsComponentDepends on Component "Records"](#destinycomponentsrecordsdestinyprofilerecordscomponentdepends-on-component-records), [Destiny.Components.Collectibles.DestinyCollectiblesComponentDepends on Component "Collectibles"](#destinycomponentscollectiblesdestinycollectiblescomponentdepends-on-component-collectibles), [Destiny.Components.Collectibles.DestinyCollectibleComponent](#destinycomponentscollectiblesdestinycollectiblecomponent), [Destiny.DestinyCollectibleStateEnumeration](#destinydestinycollectiblestateenumeration), [Destiny.Components.Collectibles.DestinyProfileCollectiblesComponentDepends on Component "Collectibles"](#destinycomponentscollectiblesdestinyprofilecollectiblescomponentdepends-on-component-collectibles), [Destiny.Components.Profiles.DestinyProfileTransitoryComponentDepends on Component "Transitory"](#destinycomponentsprofilesdestinyprofiletransitorycomponentdepends-on-component-transitory), [Destiny.Components.Profiles.DestinyProfileTransitoryPartyMember](#destinycomponentsprofilesdestinyprofiletransitorypartymember), [Destiny.DestinyPartyMemberStatesEnumeration](#destinydestinypartymemberstatesenumeration), [Destiny.Components.Profiles.DestinyProfileTransitoryCurrentActivity](#destinycomponentsprofilesdestinyprofiletransitorycurrentactivity), [Destiny.Components.Profiles.DestinyProfileTransitoryJoinability](#destinycomponentsprofilesdestinyprofiletransitoryjoinability), [Destiny.DestinyGamePrivacySettingEnumeration](#destinydestinygameprivacysettingenumeration), [Destiny.DestinyJoinClosedReasonsEnumeration](#destinydestinyjoinclosedreasonsenumeration), [Destiny.Components.Profiles.DestinyProfileTransitoryTrackingEntry](#destinycomponentsprofilesdestinyprofiletransitorytrackingentry), [Destiny.Components.Metrics.DestinyMetricsComponentDepends on Component "Metrics"](#destinycomponentsmetricsdestinymetricscomponentdepends-on-component-metrics), [Destiny.Components.Metrics.DestinyMetricComponent](#destinycomponentsmetricsdestinymetriccomponent), [Destiny.Components.StringVariables.DestinyStringVariablesComponentDepends on Component "StringVariables"](#destinycomponentsstringvariablesdestinystringvariablescomponentdepends-on-component-stringvariables), [Destiny.Components.Social.DestinySocialCommendationsComponentDepends on Component "SocialCommendations"](#destinycomponentssocialdestinysocialcommendationscomponentdepends-on-component-socialcommendations), [Destiny.Definitions.Social.DestinySocialCommendationNodeDefinition](#destinydefinitionssocialdestinysocialcommendationnodedefinition), [Destiny.Definitions.Social.DestinySocialCommendationDefinition](#destinydefinitionssocialdestinysocialcommendationdefinition), [Destiny.Entities.Characters.DestinyCharacterComponentDepends on Component "Characters"](#destinyentitiescharactersdestinycharactercomponentdepends-on-component-characters), [Destiny.DestinyRaceEnumeration](#destinydestinyraceenumeration), [Destiny.Definitions.DestinyRaceDefinition](#destinydefinitionsdestinyracedefinition), [Destiny.Components.Loadouts.DestinyLoadoutsComponentDepends on Component "CharacterLoadouts"](#destinycomponentsloadoutsdestinyloadoutscomponentdepends-on-component-characterloadouts), [Destiny.Components.Loadouts.DestinyLoadoutComponent](#destinycomponentsloadoutsdestinyloadoutcomponent), [Destiny.Components.Loadouts.DestinyLoadoutItemComponent](#destinycomponentsloadoutsdestinyloadoutitemcomponent), [Destiny.Definitions.Loadouts.DestinyLoadoutColorDefinition](#destinydefinitionsloadoutsdestinyloadoutcolordefinition), [Destiny.Definitions.Loadouts.DestinyLoadoutIconDefinition](#destinydefinitionsloadoutsdestinyloadouticondefinition), [Destiny.Definitions.Loadouts.DestinyLoadoutNameDefinition](#destinydefinitionsloadoutsdestinyloadoutnamedefinition), [Destiny.Entities.Characters.DestinyCharacterProgressionComponentDepends on Component "CharacterProgressions"](#destinyentitiescharactersdestinycharacterprogressioncomponentdepends-on-component-characterprogressions), [Destiny.Progression.DestinyFactionProgression](#destinyprogressiondestinyfactionprogression), [Destiny.Milestones.DestinyMilestone](#destinymilestonesdestinymilestone), [Destiny.Milestones.DestinyMilestoneQuest](#destinymilestonesdestinymilestonequest), [Destiny.Quests.DestinyQuestStatus](#destinyquestsdestinyqueststatus), [Destiny.Milestones.DestinyMilestoneActivity](#destinymilestonesdestinymilestoneactivity), [Destiny.Milestones.DestinyMilestoneActivityVariant](#destinymilestonesdestinymilestoneactivityvariant), [Destiny.Milestones.DestinyMilestoneActivityCompletionStatus](#destinymilestonesdestinymilestoneactivitycompletionstatus), [Destiny.Milestones.DestinyMilestoneActivityPhase](#destinymilestonesdestinymilestoneactivityphase), [Destiny.Challenges.DestinyChallengeStatus](#destinychallengesdestinychallengestatus), [Destiny.Milestones.DestinyMilestoneChallengeActivity](#destinymilestonesdestinymilestonechallengeactivity), [Destiny.Milestones.DestinyMilestoneVendor](#destinymilestonesdestinymilestonevendor), [Destiny.Milestones.DestinyMilestoneRewardCategory](#destinymilestonesdestinymilestonerewardcategory), [Destiny.Milestones.DestinyMilestoneRewardEntry](#destinymilestonesdestinymilestonerewardentry), [Destiny.Definitions.Milestones.DestinyMilestoneDefinition](#destinydefinitionsmilestonesdestinymilestonedefinition), [Destiny.Definitions.Milestones.DestinyMilestoneDisplayPreferenceEnumeration](#destinydefinitionsmilestonesdestinymilestonedisplaypreferenceenumeration), [Destiny.Definitions.Milestones.DestinyMilestoneTypeEnumeration](#destinydefinitionsmilestonesdestinymilestonetypeenumeration), [Destiny.Definitions.Milestones.DestinyMilestoneQuestDefinition](#destinydefinitionsmilestonesdestinymilestonequestdefinition), [Destiny.Definitions.Milestones.DestinyMilestoneQuestRewardsDefinition](#destinydefinitionsmilestonesdestinymilestonequestrewardsdefinition), [Destiny.Definitions.Milestones.DestinyMilestoneQuestRewardItem](#destinydefinitionsmilestonesdestinymilestonequestrewarditem), [Destiny.Definitions.Milestones.DestinyMilestoneActivityDefinition](#destinydefinitionsmilestonesdestinymilestoneactivitydefinition), [Destiny.Definitions.Milestones.DestinyMilestoneActivityVariantDefinition](#destinydefinitionsmilestonesdestinymilestoneactivityvariantdefinition), [Destiny.Definitions.Milestones.DestinyMilestoneRewardCategoryDefinition](#destinydefinitionsmilestonesdestinymilestonerewardcategorydefinition), [Destiny.Definitions.Milestones.DestinyMilestoneRewardEntryDefinition](#destinydefinitionsmilestonesdestinymilestonerewardentrydefinition), [Destiny.Definitions.Milestones.DestinyMilestoneVendorDefinition](#destinydefinitionsmilestonesdestinymilestonevendordefinition), [Destiny.Definitions.Milestones.DestinyMilestoneValueDefinition](#destinydefinitionsmilestonesdestinymilestonevaluedefinition), [Destiny.Definitions.Milestones.DestinyMilestoneChallengeActivityDefinition](#destinydefinitionsmilestonesdestinymilestonechallengeactivitydefinition), [Destiny.Definitions.Milestones.DestinyMilestoneChallengeDefinition](#destinydefinitionsmilestonesdestinymilestonechallengedefinition), [Destiny.Definitions.Milestones.DestinyMilestoneChallengeActivityGraphNodeEntry](#destinydefinitionsmilestonesdestinymilestonechallengeactivitygraphnodeentry), [Destiny.Definitions.Milestones.DestinyMilestoneChallengeActivityPhase](#destinydefinitionsmilestonesdestinymilestonechallengeactivityphase), [Destiny.Entities.Items.DestinyItemPerksComponentDepends on Component "ItemPerks"](#destinyentitiesitemsdestinyitemperkscomponentdepends-on-component-itemperks), [Destiny.Perks.DestinyPerkReference](#destinyperksdestinyperkreference), [Destiny.Artifacts.DestinyArtifactCharacterScoped](#destinyartifactsdestinyartifactcharacterscoped), [Destiny.Artifacts.DestinyArtifactTier](#destinyartifactsdestinyartifacttier), [Destiny.Artifacts.DestinyArtifactTierItem](#destinyartifactsdestinyartifacttieritem), [Destiny.Entities.Characters.DestinyCharacterRenderComponentDepends on Component "CharacterRenderData"](#destinyentitiescharactersdestinycharacterrendercomponentdepends-on-component-characterrenderdata), [Destiny.Character.DestinyCharacterCustomization](#destinycharacterdestinycharactercustomization), [Destiny.Character.DestinyCharacterPeerView](#destinycharacterdestinycharacterpeerview), [Destiny.Character.DestinyItemPeerView](#destinycharacterdestinyitempeerview), [Destiny.Entities.Characters.DestinyCharacterActivitiesComponentDepends on Component "CharacterActivities"](#destinyentitiescharactersdestinycharacteractivitiescomponentdepends-on-component-characteractivities), [Destiny.DestinyActivity](#destinydestinyactivity), [Destiny.DestinyActivityDifficultyTierEnumeration](#destinydestinyactivitydifficultytierenumeration), [Destiny.Definitions.DestinyActivityRewardMapping](#destinydefinitionsdestinyactivityrewardmapping), [Destiny.DestinyActivityRewardDisplayModeEnumeration](#destinydestinyactivityrewarddisplaymodeenumeration), [Destiny.Definitions.DestinyActivityRewardItem](#destinydefinitionsdestinyactivityrewarditem), [Destiny.Definitions.FireteamFinder.DestinyActivityInteractableReference](#destinydefinitionsfireteamfinderdestinyactivityinteractablereference), [Destiny.Definitions.Activities.DestinyActivityInteractableDefinition](#destinydefinitionsactivitiesdestinyactivityinteractabledefinition), [Destiny.Definitions.Activities.DestinyActivityInteractableEntryDefinition](#destinydefinitionsactivitiesdestinyactivityinteractableentrydefinition), [Destiny.DestinyActivityDifficultyTierCollectionComponent](#destinydestinyactivitydifficultytiercollectioncomponent), [Destiny.DestinyActivityDifficultyTierComponent](#destinydestinyactivitydifficultytiercomponent), [Destiny.DestinyActivitySkullComponent](#destinydestinyactivityskullcomponent), [Destiny.DestinyActivitySelectableSkullCollectionComponent](#destinydestinyactivityselectableskullcollectioncomponent), [Destiny.Entities.Items.DestinyItemObjectivesComponentDepends on Component "ItemObjectives"](#destinyentitiesitemsdestinyitemobjectivescomponentdepends-on-component-itemobjectives), [Destiny.Components.Records.DestinyCharacterRecordsComponentDepends on Component "Records"](#destinycomponentsrecordsdestinycharacterrecordscomponentdepends-on-component-records), [Destiny.Components.Craftables.DestinyCraftablesComponentDepends on Component "Craftables"](#destinycomponentscraftablesdestinycraftablescomponentdepends-on-component-craftables), [Destiny.Components.Craftables.DestinyCraftableComponent](#destinycomponentscraftablesdestinycraftablecomponent), [Destiny.Components.Craftables.DestinyCraftableSocketComponent](#destinycomponentscraftablesdestinycraftablesocketcomponent), [Destiny.Components.Craftables.DestinyCraftableSocketPlugComponent](#destinycomponentscraftablesdestinycraftablesocketplugcomponent), [Destiny.Entities.Items.DestinyItemInstanceComponentDepends on Component "ItemInstances"](#destinyentitiesitemsdestinyiteminstancecomponentdepends-on-component-iteminstances), [Destiny.EquipFailureReasonEnumeration](#destinyequipfailurereasonenumeration), [Destiny.Entities.Items.DestinyItemInstanceEnergy](#destinyentitiesitemsdestinyiteminstanceenergy), [Destiny.Definitions.DestinyUnlockDefinition](#destinydefinitionsdestinyunlockdefinition), [Destiny.Entities.Items.DestinyItemRenderComponentDepends on Component "ItemRenderData"](#destinyentitiesitemsdestinyitemrendercomponentdepends-on-component-itemrenderdata), [Destiny.Entities.Items.DestinyItemStatsComponentDepends on Component "ItemStats"](#destinyentitiesitemsdestinyitemstatscomponentdepends-on-component-itemstats), [Destiny.Entities.Items.DestinyItemSocketsComponentDepends on Component "ItemSockets"](#destinyentitiesitemsdestinyitemsocketscomponentdepends-on-component-itemsockets), [Destiny.Entities.Items.DestinyItemSocketState](#destinyentitiesitemsdestinyitemsocketstate), [Destiny.Components.Items.DestinyItemReusablePlugsComponentDepends on Component "ItemReusablePlugs"](#destinycomponentsitemsdestinyitemreusableplugscomponentdepends-on-component-itemreusableplugs), [Destiny.Components.Items.DestinyItemPlugObjectivesComponentDepends on Component "ItemPlugObjectives"](#destinycomponentsitemsdestinyitemplugobjectivescomponentdepends-on-component-itemplugobjectives), [Destiny.Entities.Items.DestinyItemTalentGridComponentDepends on Component "ItemTalentGrids"](#destinyentitiesitemsdestinyitemtalentgridcomponentdepends-on-component-itemtalentgrids), [Destiny.DestinyTalentNode](#destinydestinytalentnode), [Destiny.DestinyTalentNodeStateEnumeration](#destinydestinytalentnodestateenumeration), [Destiny.DestinyTalentNodeStatBlock](#destinydestinytalentnodestatblock), [Destiny.Components.Items.DestinyItemPlugComponentDepends on Component "ItemPlugStates"](#destinycomponentsitemsdestinyitemplugcomponentdepends-on-component-itemplugstates), [Destiny.Components.Inventory.DestinyCurrenciesComponentDepends on Component "CurrencyLookups"](#destinycomponentsinventorydestinycurrenciescomponentdepends-on-component-currencylookups), [Destiny.Components.Inventory.DestinyMaterialRequirementSetState](#destinycomponentsinventorydestinymaterialrequirementsetstate), [Destiny.Components.Inventory.DestinyMaterialRequirementState](#destinycomponentsinventorydestinymaterialrequirementstate), [Destiny.Responses.DestinyCharacterResponse](#destinyresponsesdestinycharacterresponse), [Destiny.Responses.DestinyItemResponse](#destinyresponsesdestinyitemresponse), [Destiny.DestinyVendorFilterEnumeration](#destinydestinyvendorfilterenumeration), [Destiny.Responses.DestinyVendorsResponse](#destinyresponsesdestinyvendorsresponse), [Destiny.Components.Vendors.DestinyVendorGroupComponentDepends on Component "Vendors"](#destinycomponentsvendorsdestinyvendorgroupcomponentdepends-on-component-vendors), [Destiny.Components.Vendors.DestinyVendorGroup](#destinycomponentsvendorsdestinyvendorgroup), [Destiny.Components.Vendors.DestinyVendorBaseComponentDepends on Component "Vendors"](#destinycomponentsvendorsdestinyvendorbasecomponentdepends-on-component-vendors), [Destiny.Entities.Vendors.DestinyVendorComponentDepends on Component "Vendors"](#destinyentitiesvendorsdestinyvendorcomponentdepends-on-component-vendors), [Destiny.Entities.Vendors.DestinyVendorCategoriesComponentDepends on Component "VendorCategories"](#destinyentitiesvendorsdestinyvendorcategoriescomponentdepends-on-component-vendorcategories), [Destiny.Entities.Vendors.DestinyVendorCategory](#destinyentitiesvendorsdestinyvendorcategory), [Destiny.Components.Vendors.DestinyVendorSaleItemBaseComponentDepends on Component "VendorSales"](#destinycomponentsvendorsdestinyvendorsaleitembasecomponentdepends-on-component-vendorsales), [Destiny.Entities.Vendors.DestinyVendorSaleItemComponentDepends on Component "VendorSales"](#destinyentitiesvendorsdestinyvendorsaleitemcomponentdepends-on-component-vendorsales), [Destiny.VendorItemStatusEnumeration](#destinyvendoritemstatusenumeration), [Destiny.DestinyUnlockStatus](#destinydestinyunlockstatus), [Destiny.DestinyVendorItemStateEnumeration](#destinydestinyvendoritemstateenumeration), [Destiny.Responses.PersonalDestinyVendorSaleItemSetComponentDepends on Component "VendorSales"](#destinyresponsespersonaldestinyvendorsaleitemsetcomponentdepends-on-component-vendorsales), [Destiny.Responses.DestinyVendorResponse](#destinyresponsesdestinyvendorresponse), [Destiny.Responses.DestinyPublicVendorsResponse](#destinyresponsesdestinypublicvendorsresponse), [Destiny.Components.Vendors.DestinyPublicVendorComponentDepends on Component "Vendors"](#destinycomponentsvendorsdestinypublicvendorcomponentdepends-on-component-vendors), [Destiny.Components.Vendors.DestinyPublicVendorSaleItemComponentDepends on Component "VendorSales"](#destinycomponentsvendorsdestinypublicvendorsaleitemcomponentdepends-on-component-vendorsales), [Destiny.Responses.PublicDestinyVendorSaleItemSetComponentDepends on Component "VendorSales"](#destinyresponsespublicdestinyvendorsaleitemsetcomponentdepends-on-component-vendorsales), [Destiny.Responses.DestinyCollectibleNodeDetailResponse](#destinyresponsesdestinycollectiblenodedetailresponse), [Destiny.Requests.Actions.DestinyActionRequest](#destinyrequestsactionsdestinyactionrequest), [Destiny.Requests.Actions.DestinyCharacterActionRequest](#destinyrequestsactionsdestinycharacteractionrequest), [Destiny.Requests.Actions.DestinyItemActionRequest](#destinyrequestsactionsdestinyitemactionrequest), [Destiny.Requests.DestinyItemTransferRequest](#destinyrequestsdestinyitemtransferrequest), [Destiny.Requests.Actions.DestinyPostmasterTransferRequest](#destinyrequestsactionsdestinypostmastertransferrequest), [Destiny.DestinyEquipItemResults](#destinydestinyequipitemresults), [Destiny.DestinyEquipItemResult](#destinydestinyequipitemresult), [Destiny.Requests.Actions.DestinyItemSetActionRequest](#destinyrequestsactionsdestinyitemsetactionrequest), [Destiny.Requests.Actions.DestinyLoadoutActionRequest](#destinyrequestsactionsdestinyloadoutactionrequest), [Destiny.Requests.Actions.DestinyLoadoutUpdateActionRequest](#destinyrequestsactionsdestinyloadoutupdateactionrequest), [Destiny.Requests.Actions.DestinyItemStateRequest](#destinyrequestsactionsdestinyitemstaterequest), [Destiny.Responses.InventoryChangedResponse](#destinyresponsesinventorychangedresponse), [Destiny.Responses.DestinyItemChangeResponse](#destinyresponsesdestinyitemchangeresponse), [Destiny.Requests.Actions.DestinyInsertPlugsActionRequest](#destinyrequestsactionsdestinyinsertplugsactionrequest), [Destiny.Requests.Actions.DestinyInsertPlugsRequestEntry](#destinyrequestsactionsdestinyinsertplugsrequestentry), [Destiny.Requests.Actions.DestinySocketArrayTypeEnumeration](#destinyrequestsactionsdestinysocketarraytypeenumeration), [Destiny.Requests.Actions.DestinyInsertPlugsFreeActionRequest](#destinyrequestsactionsdestinyinsertplugsfreeactionrequest), [Destiny.HistoricalStats.DestinyPostGameCarnageReportData](#destinyhistoricalstatsdestinypostgamecarnagereportdata), [Destiny.HistoricalStats.DestinyHistoricalStatsActivity](#destinyhistoricalstatsdestinyhistoricalstatsactivity), [Destiny.HistoricalStats.DestinyPostGameCarnageReportEntry](#destinyhistoricalstatsdestinypostgamecarnagereportentry), [Destiny.HistoricalStats.DestinyHistoricalStatsValue](#destinyhistoricalstatsdestinyhistoricalstatsvalue), [Destiny.HistoricalStats.DestinyHistoricalStatsValuePair](#destinyhistoricalstatsdestinyhistoricalstatsvaluepair), [Destiny.HistoricalStats.DestinyPlayer](#destinyhistoricalstatsdestinyplayer), [Destiny.HistoricalStats.DestinyPostGameCarnageReportExtendedData](#destinyhistoricalstatsdestinypostgamecarnagereportextendeddata), [Destiny.HistoricalStats.DestinyHistoricalWeaponStats](#destinyhistoricalstatsdestinyhistoricalweaponstats), [Destiny.HistoricalStats.DestinyPostGameCarnageReportTeamEntry](#destinyhistoricalstatsdestinypostgamecarnagereportteamentry), [Destiny.Reporting.Requests.DestinyReportOffensePgcrRequest](#destinyreportingrequestsdestinyreportoffensepgcrrequest), [Destiny.Definitions.Reporting.DestinyReportReasonCategoryDefinition](#destinydefinitionsreportingdestinyreportreasoncategorydefinition), [Destiny.Definitions.Reporting.DestinyReportReasonDefinition](#destinydefinitionsreportingdestinyreportreasondefinition), [Destiny.HistoricalStats.Definitions.DestinyHistoricalStatsDefinition](#destinyhistoricalstatsdefinitionsdestinyhistoricalstatsdefinition), [Destiny.HistoricalStats.Definitions.DestinyStatsGroupTypeEnumeration](#destinyhistoricalstatsdefinitionsdestinystatsgrouptypeenumeration), [Destiny.HistoricalStats.Definitions.PeriodType[]](#destinyhistoricalstatsdefinitionsperiodtype), [Destiny.HistoricalStats.Definitions.DestinyActivityModeType[]](#destinyhistoricalstatsdefinitionsdestinyactivitymodetype), [Destiny.HistoricalStats.Definitions.DestinyStatsCategoryTypeEnumeration](#destinyhistoricalstatsdefinitionsdestinystatscategorytypeenumeration), [Destiny.HistoricalStats.Definitions.UnitTypeEnumeration](#destinyhistoricalstatsdefinitionsunittypeenumeration), [Destiny.HistoricalStats.Definitions.DestinyStatsMergeMethodEnumeration](#destinyhistoricalstatsdefinitionsdestinystatsmergemethodenumeration), [Destiny.Definitions.DestinyMedalTierDefinition](#destinydefinitionsdestinymedaltierdefinition), [Destiny.HistoricalStats.DestinyLeaderboard](#destinyhistoricalstatsdestinyleaderboard), [Destiny.HistoricalStats.DestinyLeaderboardEntry](#destinyhistoricalstatsdestinyleaderboardentry), [Destiny.HistoricalStats.DestinyLeaderboardResults](#destinyhistoricalstatsdestinyleaderboardresults), [Destiny.HistoricalStats.DestinyClanAggregateStat](#destinyhistoricalstatsdestinyclanaggregatestat), [Destiny.Definitions.DestinyEntitySearchResult](#destinydefinitionsdestinyentitysearchresult), [Destiny.Definitions.DestinyEntitySearchResultItem](#destinydefinitionsdestinyentitysearchresultitem), [Destiny.HistoricalStats.Definitions.PeriodTypeEnumeration](#destinyhistoricalstatsdefinitionsperiodtypeenumeration), [Destiny.HistoricalStats.DestinyHistoricalStatsByPeriod](#destinyhistoricalstatsdestinyhistoricalstatsbyperiod), [Destiny.HistoricalStats.DestinyHistoricalStatsPeriodGroup](#destinyhistoricalstatsdestinyhistoricalstatsperiodgroup), [Destiny.HistoricalStats.DestinyHistoricalStatsResults](#destinyhistoricalstatsdestinyhistoricalstatsresults), [Destiny.HistoricalStats.DestinyHistoricalStatsAccountResult](#destinyhistoricalstatsdestinyhistoricalstatsaccountresult), [Destiny.HistoricalStats.DestinyHistoricalStatsWithMerged](#destinyhistoricalstatsdestinyhistoricalstatswithmerged), [Destiny.HistoricalStats.DestinyHistoricalStatsPerCharacter](#destinyhistoricalstatsdestinyhistoricalstatspercharacter), [Destiny.HistoricalStats.DestinyActivityHistoryResults](#destinyhistoricalstatsdestinyactivityhistoryresults), [Destiny.HistoricalStats.DestinyHistoricalWeaponStatsData](#destinyhistoricalstatsdestinyhistoricalweaponstatsdata), [Destiny.HistoricalStats.DestinyAggregateActivityResults](#destinyhistoricalstatsdestinyaggregateactivityresults), [Destiny.HistoricalStats.DestinyAggregateActivityStats](#destinyhistoricalstatsdestinyaggregateactivitystats), [Destiny.Milestones.DestinyMilestoneContent](#destinymilestonesdestinymilestonecontent), [Destiny.Milestones.DestinyMilestoneContentItemCategory](#destinymilestonesdestinymilestonecontentitemcategory), [Destiny.Milestones.DestinyPublicMilestone](#destinymilestonesdestinypublicmilestone), [Destiny.Milestones.DestinyPublicMilestoneQuest](#destinymilestonesdestinypublicmilestonequest), [Destiny.Milestones.DestinyPublicMilestoneActivity](#destinymilestonesdestinypublicmilestoneactivity), [Destiny.Milestones.DestinyPublicMilestoneActivityVariant](#destinymilestonesdestinypublicmilestoneactivityvariant), [Destiny.Milestones.DestinyPublicMilestoneChallenge](#destinymilestonesdestinypublicmilestonechallenge), [Destiny.Milestones.DestinyPublicMilestoneChallengeActivity](#destinymilestonesdestinypublicmilestonechallengeactivity), [Destiny.Milestones.DestinyPublicMilestoneVendor](#destinymilestonesdestinypublicmilestonevendor), [Destiny.Advanced.AwaInitializeResponse](#destinyadvancedawainitializeresponse), [Destiny.Advanced.AwaPermissionRequested](#destinyadvancedawapermissionrequested), [Destiny.Advanced.AwaTypeEnumeration](#destinyadvancedawatypeenumeration), [Destiny.Advanced.AwaUserResponse](#destinyadvancedawauserresponse), [Destiny.Advanced.AwaUserSelectionEnumeration](#destinyadvancedawauserselectionenumeration), [Destiny.Advanced.AwaAuthorizationResult](#destinyadvancedawaauthorizationresult), [Destiny.Advanced.AwaResponseReasonEnumeration](#destinyadvancedawaresponsereasonenumeration), [Destiny.Activities.DestinyPublicActivityStatus](#destinyactivitiesdestinypublicactivitystatus), [Destiny.Definitions.Common.DestinyGlobalConstantsDefinition](#destinydefinitionscommondestinyglobalconstantsdefinition), [Destiny.Definitions.Common.DestinyPathfinderConstantsDefinition](#destinydefinitionscommondestinypathfinderconstantsdefinition), [Destiny.Definitions.Common.DestinyRewardPassRankSealImages](#destinydefinitionscommondestinyrewardpassranksealimages), [Destiny.Definitions.Common.DestinySeasonalHubRankIconImages](#destinydefinitionscommondestinyseasonalhubrankiconimages), [Destiny.Definitions.Inventory.DestinyItemFilterDefinition](#destinydefinitionsinventorydestinyitemfilterdefinition), [Destiny.Definitions.Loadouts.DestinyLoadoutConstantsDefinition](#destinydefinitionsloadoutsdestinyloadoutconstantsdefinition), [Destiny.Definitions.GuardianRanks.DestinyGuardianRankConstantsDefinition](#destinydefinitionsguardianranksdestinyguardianrankconstantsdefinition), [Destiny.Definitions.GuardianRanks.DestinyGuardianRankIconBackgroundsDefinition](#destinydefinitionsguardianranksdestinyguardianrankiconbackgroundsdefinition), [Destiny.Definitions.FireteamFinder.DestinyFireteamFinderConstantsDefinition](#destinydefinitionsfireteamfinderdestinyfireteamfinderconstantsdefinition), [Destiny.Definitions.FireteamFinder.DestinyFireteamFinderActivityGraphDefinition](#destinydefinitionsfireteamfinderdestinyfireteamfinderactivitygraphdefinition), [Destiny.Definitions.FireteamFinder.DestinyActivityGraphReference](#destinydefinitionsfireteamfinderdestinyactivitygraphreference), [Destiny.DestinyActivityTreeTypeEnumeration](#destinydestinyactivitytreetypeenumeration), [Destiny.DestinyActivityTreeChildSortModeEnumeration](#destinydestinyactivitytreechildsortmodeenumeration), [Destiny.Definitions.FireteamFinder.DestinyFireteamFinderActivitySetDefinition](#destinydefinitionsfireteamfinderdestinyfireteamfinderactivitysetdefinition), [Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionDefinition](#destinydefinitionsfireteamfinderdestinyfireteamfinderoptiondefinition), [Destiny.FireteamFinderCodeOptionTypeEnumeration](#destinyfireteamfindercodeoptiontypeenumeration), [Destiny.FireteamFinderOptionAvailabilityEnumeration](#destinyfireteamfinderoptionavailabilityenumeration), [Destiny.FireteamFinderOptionVisibilityEnumeration](#destinyfireteamfinderoptionvisibilityenumeration), [Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionCreatorSettings](#destinydefinitionsfireteamfinderdestinyfireteamfinderoptioncreatorsettings), [Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionSettingsControl](#destinydefinitionsfireteamfinderdestinyfireteamfinderoptionsettingscontrol), [Destiny.FireteamFinderOptionControlTypeEnumeration](#destinyfireteamfinderoptioncontroltypeenumeration), [Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionSearcherSettings](#destinydefinitionsfireteamfinderdestinyfireteamfinderoptionsearchersettings), [Destiny.FireteamFinderOptionSearchFilterTypeEnumeration](#destinyfireteamfinderoptionsearchfiltertypeenumeration), [Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionValues](#destinydefinitionsfireteamfinderdestinyfireteamfinderoptionvalues), [Destiny.FireteamFinderOptionDisplayFormatEnumeration](#destinyfireteamfinderoptiondisplayformatenumeration), [Destiny.FireteamFinderOptionValueProviderTypeEnumeration](#destinyfireteamfinderoptionvalueprovidertypeenumeration), [Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionValueDefinition](#destinydefinitionsfireteamfinderdestinyfireteamfinderoptionvaluedefinition), [Destiny.FireteamFinderOptionValueFlagsEnumeration](#destinyfireteamfinderoptionvalueflagsenumeration), [Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionGroupDefinition](#destinydefinitionsfireteamfinderdestinyfireteamfinderoptiongroupdefinition), [Destiny.Definitions.FireteamFinder.DestinyFireteamFinderLabelDefinition](#destinydefinitionsfireteamfinderdestinyfireteamfinderlabeldefinition), [Destiny.FireteamFinderLabelFieldTypeEnumeration](#destinyfireteamfinderlabelfieldtypeenumeration), [Destiny.Definitions.FireteamFinder.DestinyFireteamFinderLabelGroupDefinition](#destinydefinitionsfireteamfinderdestinyfireteamfinderlabelgroupdefinition), [Destiny.Definitions.Items.DestinyInventoryItemConstantsDefinition](#destinydefinitionsitemsdestinyinventoryitemconstantsdefinition)
- **Entities** (1): [Entities.EntityActionResult](#entitiesentityactionresult)
- **Exceptions** (1): [Exceptions.PlatformErrorCodesEnumeration](#exceptionsplatformerrorcodesenumeration)
- **Fireteam** (9): [Fireteam.FireteamDateRangeEnumeration](#fireteamfireteamdaterangeenumeration), [Fireteam.FireteamPlatformEnumeration](#fireteamfireteamplatformenumeration), [Fireteam.FireteamPublicSearchOptionEnumeration](#fireteamfireteampublicsearchoptionenumeration), [Fireteam.FireteamSlotSearchEnumeration](#fireteamfireteamslotsearchenumeration), [Fireteam.FireteamSummary](#fireteamfireteamsummary), [Fireteam.FireteamResponse](#fireteamfireteamresponse), [Fireteam.FireteamMember](#fireteamfireteammember), [Fireteam.FireteamUserInfoCard](#fireteamfireteamuserinfocard), [Fireteam.FireteamPlatformInviteResultEnumeration](#fireteamfireteamplatforminviteresultenumeration)
- **Forum** (14): [Forum.ForumTopicsCategoryFiltersEnumEnumeration](#forumforumtopicscategoryfiltersenumenumeration), [Forum.ForumTopicsQuickDateEnumEnumeration](#forumforumtopicsquickdateenumenumeration), [Forum.ForumTopicsSortEnumEnumeration](#forumforumtopicssortenumenumeration), [Forum.PostResponse](#forumpostresponse), [Forum.ForumMediaTypeEnumeration](#forumforummediatypeenumeration), [Forum.ForumPostPopularityEnumeration](#forumforumpostpopularityenumeration), [Forum.PostSearchResponse](#forumpostsearchresponse), [Forum.PollResponse](#forumpollresponse), [Forum.PollResult](#forumpollresult), [Forum.ForumRecruitmentDetail](#forumforumrecruitmentdetail), [Forum.ForumRecruitmentIntensityLabelEnumeration](#forumforumrecruitmentintensitylabelenumeration), [Forum.ForumRecruitmentToneLabelEnumeration](#forumforumrecruitmenttonelabelenumeration), [Forum.ForumPostSortEnumEnumeration](#forumforumpostsortenumenumeration), [Forum.CommunityContentSortModeEnumeration](#forumcommunitycontentsortmodeenumeration)
- **Forums** (2): [Forums.ForumPostCategoryEnumsEnumeration](#forumsforumpostcategoryenumsenumeration), [Forums.ForumFlagsEnumEnumeration](#forumsforumflagsenumenumeration)
- **GroupsV2** (48): [GroupsV2.GroupUserInfoCard](#groupsv2groupuserinfocard), [GroupsV2.GroupResponse](#groupsv2groupresponse), [GroupsV2.GroupV2](#groupsv2groupv2), [GroupsV2.GroupTypeEnumeration](#groupsv2grouptypeenumeration), [GroupsV2.ChatSecuritySettingEnumeration](#groupsv2chatsecuritysettingenumeration), [GroupsV2.GroupHomepageEnumeration](#groupsv2grouphomepageenumeration), [GroupsV2.MembershipOptionEnumeration](#groupsv2membershipoptionenumeration), [GroupsV2.GroupPostPublicityEnumeration](#groupsv2grouppostpublicityenumeration), [GroupsV2.GroupFeatures](#groupsv2groupfeatures), [GroupsV2.CapabilitiesEnumeration](#groupsv2capabilitiesenumeration), [GroupsV2.HostGuidedGamesPermissionLevelEnumeration](#groupsv2hostguidedgamespermissionlevelenumeration), [GroupsV2.RuntimeGroupMemberTypeEnumeration](#groupsv2runtimegroupmembertypeenumeration), [GroupsV2.GroupV2ClanInfo](#groupsv2groupv2claninfo), [GroupsV2.ClanBanner](#groupsv2clanbanner), [GroupsV2.GroupV2ClanInfoAndInvestment](#groupsv2groupv2claninfoandinvestment), [GroupsV2.GroupUserBase](#groupsv2groupuserbase), [GroupsV2.GroupMember](#groupsv2groupmember), [GroupsV2.GroupAllianceStatusEnumeration](#groupsv2groupalliancestatusenumeration), [GroupsV2.GroupPotentialMember](#groupsv2grouppotentialmember), [GroupsV2.GroupPotentialMemberStatusEnumeration](#groupsv2grouppotentialmemberstatusenumeration), [GroupsV2.GroupDateRangeEnumeration](#groupsv2groupdaterangeenumeration), [GroupsV2.GroupV2Card](#groupsv2groupv2card), [GroupsV2.GroupSearchResponse](#groupsv2groupsearchresponse), [GroupsV2.GroupQuery](#groupsv2groupquery), [GroupsV2.GroupSortByEnumeration](#groupsv2groupsortbyenumeration), [GroupsV2.GroupMemberCountFilterEnumeration](#groupsv2groupmembercountfilterenumeration), [GroupsV2.GroupNameSearchRequest](#groupsv2groupnamesearchrequest), [GroupsV2.GroupOptionalConversation](#groupsv2groupoptionalconversation), [GroupsV2.GroupEditAction](#groupsv2groupeditaction), [GroupsV2.GroupOptionsEditAction](#groupsv2groupoptionseditaction), [GroupsV2.GroupOptionalConversationAddRequest](#groupsv2groupoptionalconversationaddrequest), [GroupsV2.GroupOptionalConversationEditRequest](#groupsv2groupoptionalconversationeditrequest), [GroupsV2.GroupMemberLeaveResult](#groupsv2groupmemberleaveresult), [GroupsV2.GroupBanRequest](#groupsv2groupbanrequest), [GroupsV2.GroupBan](#groupsv2groupban), [GroupsV2.GroupEditHistory](#groupsv2groupedithistory), [GroupsV2.GroupMemberApplication](#groupsv2groupmemberapplication), [GroupsV2.GroupApplicationResolveStateEnumeration](#groupsv2groupapplicationresolvestateenumeration), [GroupsV2.GroupApplicationRequest](#groupsv2groupapplicationrequest), [GroupsV2.GroupApplicationListRequest](#groupsv2groupapplicationlistrequest), [GroupsV2.GroupsForMemberFilterEnumeration](#groupsv2groupsformemberfilterenumeration), [GroupsV2.GroupMembershipBase](#groupsv2groupmembershipbase), [GroupsV2.GroupMembership](#groupsv2groupmembership), [GroupsV2.GroupMembershipSearchResponse](#groupsv2groupmembershipsearchresponse), [GroupsV2.GetGroupsForMemberResponse](#groupsv2getgroupsformemberresponse), [GroupsV2.GroupPotentialMembership](#groupsv2grouppotentialmembership), [GroupsV2.GroupPotentialMembershipSearchResponse](#groupsv2grouppotentialmembershipsearchresponse), [GroupsV2.GroupApplicationResponse](#groupsv2groupapplicationresponse)
- **Ignores** (3): [Ignores.IgnoreResponse](#ignoresignoreresponse), [Ignores.IgnoreStatusEnumeration](#ignoresignorestatusenumeration), [Ignores.IgnoreLengthEnumeration](#ignoresignorelengthenumeration)
- **Interpolation** (2): [Interpolation.InterpolationPoint](#interpolationinterpolationpoint), [Interpolation.InterpolationPointFloat](#interpolationinterpolationpointfloat)
- **Links** (1): [Links.HyperlinkReference](#linkshyperlinkreference)
- **Queries** (2): [Queries.SearchResult](#queriessearchresult), [Queries.PagedQuery](#queriespagedquery)
- **Social** (9): [Social.Friends.BungieFriendListResponse](#socialfriendsbungiefriendlistresponse), [Social.Friends.BungieFriend](#socialfriendsbungiefriend), [Social.Friends.PresenceStatusEnumeration](#socialfriendspresencestatusenumeration), [Social.Friends.PresenceOnlineStateFlagsEnumeration](#socialfriendspresenceonlinestateflagsenumeration), [Social.Friends.FriendRelationshipStateEnumeration](#socialfriendsfriendrelationshipstateenumeration), [Social.Friends.BungieFriendRequestListResponse](#socialfriendsbungiefriendrequestlistresponse), [Social.Friends.PlatformFriendTypeEnumeration](#socialfriendsplatformfriendtypeenumeration), [Social.Friends.PlatformFriendResponse](#socialfriendsplatformfriendresponse), [Social.Friends.PlatformFriend](#socialfriendsplatformfriend)
- **Streaming** (1): [Streaming.DropStateEnumEnumeration](#streamingdropstateenumenumeration)
- **Tags** (1): [Tags.Models.Contracts.TagResponse](#tagsmodelscontractstagresponse)
- **Tokens** (10): [Tokens.PartnerOfferClaimRequest](#tokenspartnerofferclaimrequest), [Tokens.PartnerOfferSkuHistoryResponse](#tokenspartnerofferskuhistoryresponse), [Tokens.PartnerOfferHistoryResponse](#tokenspartnerofferhistoryresponse), [Tokens.PartnerRewardHistoryResponse](#tokenspartnerrewardhistoryresponse), [Tokens.TwitchDropHistoryResponse](#tokenstwitchdrophistoryresponse), [Tokens.BungieRewardDisplay](#tokensbungierewarddisplay), [Tokens.UserRewardAvailabilityModel](#tokensuserrewardavailabilitymodel), [Tokens.RewardAvailabilityModel](#tokensrewardavailabilitymodel), [Tokens.CollectibleDefinitions](#tokenscollectibledefinitions), [Tokens.RewardDisplayProperties](#tokensrewarddisplayproperties)
- **Trending** (11): [Trending.TrendingCategories](#trendingtrendingcategories), [Trending.TrendingCategory](#trendingtrendingcategory), [Trending.TrendingEntry](#trendingtrendingentry), [Trending.TrendingEntryTypeEnumeration](#trendingtrendingentrytypeenumeration), [Trending.TrendingDetail](#trendingtrendingdetail), [Trending.TrendingEntryNews](#trendingtrendingentrynews), [Trending.TrendingEntrySupportArticle](#trendingtrendingentrysupportarticle), [Trending.TrendingEntryDestinyItem](#trendingtrendingentrydestinyitem), [Trending.TrendingEntryDestinyActivity](#trendingtrendingentrydestinyactivity), [Trending.TrendingEntryDestinyRitual](#trendingtrendingentrydestinyritual), [Trending.TrendingEntryCommunityCreation](#trendingtrendingentrycommunitycreation)
- **User** (20): [User.UserMembership](#userusermembership), [User.CrossSaveUserMembership](#usercrosssaveusermembership), [User.UserInfoCard](#useruserinfocard), [User.GeneralUser](#usergeneraluser), [User.UserToUserContext](#userusertousercontext), [User.Models.GetCredentialTypesForAccountResponse](#usermodelsgetcredentialtypesforaccountresponse), [User.UserMembershipData](#userusermembershipdata), [User.HardLinkedUserMembership](#userhardlinkedusermembership), [User.UserSearchResponse](#userusersearchresponse), [User.UserSearchResponseDetail](#userusersearchresponsedetail), [User.UserSearchPrefixRequest](#userusersearchprefixrequest), [User.ExactSearchRequest](#userexactsearchrequest), [User.EmailSettings](#useremailsettings), [User.EmailOptInDefinition](#useremailoptindefinition), [User.OptInFlagsEnumeration](#useroptinflagsenumeration), [User.EmailSubscriptionDefinition](#useremailsubscriptiondefinition), [User.EMailSettingLocalization](#useremailsettinglocalization), [User.EMailSettingSubscriptionLocalization](#useremailsettingsubscriptionlocalization), [User.EmailViewDefinition](#useremailviewdefinition), [User.EmailViewDefinitionSetting](#useremailviewdefinitionsetting)

## Entities (types & enums)

### Namespace: (root types)

#### BungieMembershipTypeEnumeration

**Enum** (`int32`)

The types of membership the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.MembershipType.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `TigerXbox` | 1 | — |
| `TigerPsn` | 2 | — |
| `TigerSteam` | 3 | — |
| `TigerBlizzard` | 4 | — |
| `TigerStadia` | 5 | — |
| `TigerEgs` | 6 | — |
| `TigerDemon` | 10 | — |
| `GoliathGame` | 20 | — |
| `BungieNext` | 254 | — |
| `All` | -1 | "All" is only valid for searching capabilities: you need to pass the actual matching BungieMembershipType for any query where you pass a known membershipId. |

#### BungieCredentialTypeEnumeration

**Enum** (`byte`)

The types of credentials the Accounts system supports. This is the external facing enum used in place of the internal-only Bungie.SharedDefinitions.CredentialType.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Xuid` | 1 | — |
| `Psnid` | 2 | — |
| `Wlid` | 3 | — |
| `Fake` | 4 | — |
| `Facebook` | 5 | — |
| `Google` | 8 | — |
| `Windows` | 9 | — |
| `DemonId` | 10 | — |
| `SteamId` | 12 | — |
| `BattleNetId` | 14 | — |
| `StadiaId` | 16 | — |
| `TwitchId` | 18 | — |
| `EgsId` | 20 | — |

#### SearchResultOfContentItemPublicContract

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;Content.ContentItemPublicContract&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### SearchResultOfPostResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;Forum.PostResponse&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### BungieMembershipType[]

**Type:** object

Type alias: `array<int32>`

#### SearchResultOfGroupV2Card

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;GroupsV2.GroupV2Card&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### SearchResultOfGroupMember

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;GroupsV2.GroupMember&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### SearchResultOfGroupBan

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;GroupsV2.GroupBan&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### SearchResultOfGroupEditHistory

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;GroupsV2.GroupEditHistory&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### SearchResultOfGroupMemberApplication

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;GroupsV2.GroupMemberApplication&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### SearchResultOfGroupMembership

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;GroupsV2.GroupMembership&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### SearchResultOfGroupPotentialMembership

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;GroupsV2.GroupPotentialMembership&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### SingleComponentResponseOfDestinyVendorReceiptsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Profiles.DestinyVendorReceiptsComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyInventoryComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Inventory.DestinyInventoryComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyProfileComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Profiles.DestinyProfileComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyPlatformSilverComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Inventory.DestinyPlatformSilverComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyKiosksComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Kiosks.DestinyKiosksComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyPlugSetsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.PlugSets.DestinyPlugSetsComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyProfileProgressionComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Profiles.DestinyProfileProgressionComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyPresentationNodesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Presentation.DestinyPresentationNodesComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyProfileRecordsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Records.DestinyProfileRecordsComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyProfileCollectiblesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Collectibles.DestinyProfileCollectiblesComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyProfileTransitoryComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Profiles.DestinyProfileTransitoryComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyMetricsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Metrics.DestinyMetricsComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyStringVariablesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.StringVariables.DestinyStringVariablesComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinySocialCommendationsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Social.DestinySocialCommendationsComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyCharacterComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Entities.Characters.DestinyCharacterComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyInventoryComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Entities.Inventory.DestinyInventoryComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyLoadoutsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Components.Loadouts.DestinyLoadoutsComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyCharacterProgressionComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Entities.Characters.DestinyCharacterProgressionComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyCharacterRenderComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Entities.Characters.DestinyCharacterRenderComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyCharacterActivitiesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Entities.Characters.DestinyCharacterActivitiesComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyKiosksComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Components.Kiosks.DestinyKiosksComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyPlugSetsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Components.PlugSets.DestinyPlugSetsComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DestinyBaseItemComponentSetOfuint32

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `objectives` | DictionaryComponentResponseOfuint32AndDestinyItemObjectivesComponent | — |
| `perks` | DictionaryComponentResponseOfuint32AndDestinyItemPerksComponent | — |

#### DictionaryComponentResponseOfuint32AndDestinyItemObjectivesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;uint32, Destiny.Entities.Items.DestinyItemObjectivesComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfuint32AndDestinyItemPerksComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;uint32, Destiny.Entities.Items.DestinyItemPerksComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyPresentationNodesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Components.Presentation.DestinyPresentationNodesComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyCharacterRecordsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Components.Records.DestinyCharacterRecordsComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyCollectiblesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Components.Collectibles.DestinyCollectiblesComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyStringVariablesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Components.StringVariables.DestinyStringVariablesComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyCraftablesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Components.Craftables.DestinyCraftablesComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DestinyBaseItemComponentSetOfint64

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `objectives` | DictionaryComponentResponseOfint64AndDestinyItemObjectivesComponent | — |
| `perks` | DictionaryComponentResponseOfint64AndDestinyItemPerksComponent | — |

#### DictionaryComponentResponseOfint64AndDestinyItemObjectivesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Entities.Items.DestinyItemObjectivesComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyItemPerksComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Entities.Items.DestinyItemPerksComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DestinyItemComponentSetOfint64

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `instances` | DictionaryComponentResponseOfint64AndDestinyItemInstanceComponent | — |
| `renderData` | DictionaryComponentResponseOfint64AndDestinyItemRenderComponent | — |
| `stats` | DictionaryComponentResponseOfint64AndDestinyItemStatsComponent | — |
| `sockets` | DictionaryComponentResponseOfint64AndDestinyItemSocketsComponent | — |
| `reusablePlugs` | DictionaryComponentResponseOfint64AndDestinyItemReusablePlugsComponent | — |
| `plugObjectives` | DictionaryComponentResponseOfint64AndDestinyItemPlugObjectivesComponent | — |
| `talentGrids` | DictionaryComponentResponseOfint64AndDestinyItemTalentGridComponent | — |
| `plugStates` | DictionaryComponentResponseOfuint32AndDestinyItemPlugComponent | — |
| `objectives` | DictionaryComponentResponseOfint64AndDestinyItemObjectivesComponent | — |
| `perks` | DictionaryComponentResponseOfint64AndDestinyItemPerksComponent | — |

#### DictionaryComponentResponseOfint64AndDestinyItemInstanceComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Entities.Items.DestinyItemInstanceComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyItemRenderComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Entities.Items.DestinyItemRenderComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyItemStatsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Entities.Items.DestinyItemStatsComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyItemSocketsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Entities.Items.DestinyItemSocketsComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyItemReusablePlugsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Components.Items.DestinyItemReusablePlugsComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyItemPlugObjectivesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Components.Items.DestinyItemPlugObjectivesComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyItemTalentGridComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Entities.Items.DestinyItemTalentGridComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfuint32AndDestinyItemPlugComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;uint32, Destiny.Components.Items.DestinyItemPlugComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint64AndDestinyCurrenciesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int64, Destiny.Components.Inventory.DestinyCurrenciesComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyCharacterComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Characters.DestinyCharacterComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyCharacterProgressionComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Characters.DestinyCharacterProgressionComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyCharacterRenderComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Characters.DestinyCharacterRenderComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyCharacterActivitiesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Characters.DestinyCharacterActivitiesComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyLoadoutsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Loadouts.DestinyLoadoutsComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyCharacterRecordsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Records.DestinyCharacterRecordsComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyCollectiblesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Collectibles.DestinyCollectiblesComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyCurrenciesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Inventory.DestinyCurrenciesComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyItemComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Items.DestinyItemComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyItemInstanceComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Items.DestinyItemInstanceComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyItemObjectivesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Items.DestinyItemObjectivesComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyItemPerksComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Items.DestinyItemPerksComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyItemRenderComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Items.DestinyItemRenderComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyItemStatsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Items.DestinyItemStatsComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyItemTalentGridComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Items.DestinyItemTalentGridComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyItemSocketsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Items.DestinyItemSocketsComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyItemReusablePlugsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Items.DestinyItemReusablePlugsComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyItemPlugObjectivesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Items.DestinyItemPlugObjectivesComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyVendorGroupComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Components.Vendors.DestinyVendorGroupComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfuint32AndDestinyVendorComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;uint32, Destiny.Entities.Vendors.DestinyVendorComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfuint32AndDestinyVendorCategoriesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;uint32, Destiny.Entities.Vendors.DestinyVendorCategoriesComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DestinyVendorSaleItemSetComponentOfDestinyVendorSaleItemComponentDepends on Component "VendorSales"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `saleItems` | Mapping&lt;int32, Destiny.Entities.Vendors.DestinyVendorSaleItemComponent&gt; | — |

#### DictionaryComponentResponseOfuint32AndPersonalDestinyVendorSaleItemSetComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;uint32, Destiny.Responses.PersonalDestinyVendorSaleItemSetComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DestinyBaseItemComponentSetOfint32

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `objectives` | DictionaryComponentResponseOfint32AndDestinyItemObjectivesComponent | — |
| `perks` | DictionaryComponentResponseOfint32AndDestinyItemPerksComponent | — |

#### DictionaryComponentResponseOfint32AndDestinyItemObjectivesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int32, Destiny.Entities.Items.DestinyItemObjectivesComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint32AndDestinyItemPerksComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int32, Destiny.Entities.Items.DestinyItemPerksComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DestinyItemComponentSetOfint32

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `instances` | DictionaryComponentResponseOfint32AndDestinyItemInstanceComponent | — |
| `renderData` | DictionaryComponentResponseOfint32AndDestinyItemRenderComponent | — |
| `stats` | DictionaryComponentResponseOfint32AndDestinyItemStatsComponent | — |
| `sockets` | DictionaryComponentResponseOfint32AndDestinyItemSocketsComponent | — |
| `reusablePlugs` | DictionaryComponentResponseOfint32AndDestinyItemReusablePlugsComponent | — |
| `plugObjectives` | DictionaryComponentResponseOfint32AndDestinyItemPlugObjectivesComponent | — |
| `talentGrids` | DictionaryComponentResponseOfint32AndDestinyItemTalentGridComponent | — |
| `plugStates` | DictionaryComponentResponseOfuint32AndDestinyItemPlugComponent | — |
| `objectives` | DictionaryComponentResponseOfint32AndDestinyItemObjectivesComponent | — |
| `perks` | DictionaryComponentResponseOfint32AndDestinyItemPerksComponent | — |

#### DictionaryComponentResponseOfint32AndDestinyItemInstanceComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int32, Destiny.Entities.Items.DestinyItemInstanceComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint32AndDestinyItemRenderComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int32, Destiny.Entities.Items.DestinyItemRenderComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint32AndDestinyItemStatsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int32, Destiny.Entities.Items.DestinyItemStatsComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint32AndDestinyItemSocketsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int32, Destiny.Entities.Items.DestinyItemSocketsComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint32AndDestinyItemReusablePlugsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int32, Destiny.Components.Items.DestinyItemReusablePlugsComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint32AndDestinyItemPlugObjectivesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int32, Destiny.Components.Items.DestinyItemPlugObjectivesComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint32AndDestinyItemTalentGridComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int32, Destiny.Entities.Items.DestinyItemTalentGridComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DestinyVendorItemComponentSetOfint32

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `itemComponents` | DictionaryComponentResponseOfint32AndDestinyItemComponent | — |
| `instances` | DictionaryComponentResponseOfint32AndDestinyItemInstanceComponent | — |
| `renderData` | DictionaryComponentResponseOfint32AndDestinyItemRenderComponent | — |
| `stats` | DictionaryComponentResponseOfint32AndDestinyItemStatsComponent | — |
| `sockets` | DictionaryComponentResponseOfint32AndDestinyItemSocketsComponent | — |
| `reusablePlugs` | DictionaryComponentResponseOfint32AndDestinyItemReusablePlugsComponent | — |
| `plugObjectives` | DictionaryComponentResponseOfint32AndDestinyItemPlugObjectivesComponent | — |
| `talentGrids` | DictionaryComponentResponseOfint32AndDestinyItemTalentGridComponent | — |
| `plugStates` | DictionaryComponentResponseOfuint32AndDestinyItemPlugComponent | — |
| `objectives` | DictionaryComponentResponseOfint32AndDestinyItemObjectivesComponent | — |
| `perks` | DictionaryComponentResponseOfint32AndDestinyItemPerksComponent | — |

#### DictionaryComponentResponseOfint32AndDestinyItemComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int32, Destiny.Entities.Items.DestinyItemComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyVendorComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Vendors.DestinyVendorComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SingleComponentResponseOfDestinyVendorCategoriesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Destiny.Entities.Vendors.DestinyVendorCategoriesComponent | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfint32AndDestinyVendorSaleItemComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;int32, Destiny.Entities.Vendors.DestinyVendorSaleItemComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfuint32AndDestinyPublicVendorComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;uint32, Destiny.Components.Vendors.DestinyPublicVendorComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DestinyVendorSaleItemSetComponentOfDestinyPublicVendorSaleItemComponentDepends on Component "VendorSales"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `saleItems` | Mapping&lt;int32, Destiny.Components.Vendors.DestinyPublicVendorSaleItemComponent&gt; | — |

#### DictionaryComponentResponseOfuint32AndPublicDestinyVendorSaleItemSetComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;uint32, Destiny.Responses.PublicDestinyVendorSaleItemSetComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DestinyItemComponentSetOfuint32

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `instances` | DictionaryComponentResponseOfuint32AndDestinyItemInstanceComponent | — |
| `renderData` | DictionaryComponentResponseOfuint32AndDestinyItemRenderComponent | — |
| `stats` | DictionaryComponentResponseOfuint32AndDestinyItemStatsComponent | — |
| `sockets` | DictionaryComponentResponseOfuint32AndDestinyItemSocketsComponent | — |
| `reusablePlugs` | DictionaryComponentResponseOfuint32AndDestinyItemReusablePlugsComponent | — |
| `plugObjectives` | DictionaryComponentResponseOfuint32AndDestinyItemPlugObjectivesComponent | — |
| `talentGrids` | DictionaryComponentResponseOfuint32AndDestinyItemTalentGridComponent | — |
| `plugStates` | DictionaryComponentResponseOfuint32AndDestinyItemPlugComponent | — |
| `objectives` | DictionaryComponentResponseOfuint32AndDestinyItemObjectivesComponent | — |
| `perks` | DictionaryComponentResponseOfuint32AndDestinyItemPerksComponent | — |

#### DictionaryComponentResponseOfuint32AndDestinyItemInstanceComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;uint32, Destiny.Entities.Items.DestinyItemInstanceComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfuint32AndDestinyItemRenderComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;uint32, Destiny.Entities.Items.DestinyItemRenderComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfuint32AndDestinyItemStatsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;uint32, Destiny.Entities.Items.DestinyItemStatsComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfuint32AndDestinyItemSocketsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;uint32, Destiny.Entities.Items.DestinyItemSocketsComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfuint32AndDestinyItemReusablePlugsComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;uint32, Destiny.Components.Items.DestinyItemReusablePlugsComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfuint32AndDestinyItemPlugObjectivesComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;uint32, Destiny.Components.Items.DestinyItemPlugObjectivesComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### DictionaryComponentResponseOfuint32AndDestinyItemTalentGridComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `data` | Mapping&lt;uint32, Destiny.Entities.Items.DestinyItemTalentGridComponent&gt; | — |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### SearchResultOfDestinyEntitySearchResultItem

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;Destiny.Definitions.DestinyEntitySearchResultItem&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### SearchResultOfTrendingEntry

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;Trending.TrendingEntry&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### SearchResultOfFireteamSummary

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;Fireteam.FireteamSummary&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### SearchResultOfFireteamResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;Fireteam.FireteamResponse&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### GlobalAlert

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `AlertKey` | string | — |
| `AlertHtml` | string | — |
| `AlertTimestamp` | date-time | — |
| `AlertLink` | string | — |
| `AlertLevel` | int32 | — |
| `AlertType` | int32 | — |
| `StreamInfo` | StreamInfo | — |

#### GlobalAlertLevelEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Unknown` | 0 | — |
| `Blue` | 1 | — |
| `Yellow` | 2 | — |
| `Red` | 3 | — |

#### GlobalAlertTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `GlobalAlert` | 0 | — |
| `StreamingAlert` | 1 | — |

#### StreamInfo

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `ChannelName` | string | — |

### Namespace: Applications

#### Applications.ApplicationScopesEnumeration

**Enum** (`int64`)

| Value | # | Description |
| --- | --- | --- |
| `ReadBasicUserProfile` | 1 | Read basic user profile information such as the user's handle, avatar icon, etc. |
| `ReadGroups` | 2 | Read Group/Clan Forums, Wall, and Members for groups and clans that the user has joined. |
| `WriteGroups` | 4 | Write Group/Clan Forums, Wall, and Members for groups and clans that the user has joined. |
| `AdminGroups` | 8 | Administer Group/Clan Forums, Wall, and Members for groups and clans that the user is a founder or an administrator. |
| `BnetWrite` | 16 | Create new groups, clans, and forum posts, along with other actions that are reserved for Bungie.net elevated scope: not meant to be used by third party applications. |
| `MoveEquipDestinyItems` | 32 | Move or equip Destiny items |
| `ReadDestinyInventoryAndVault` | 64 | Read Destiny 1 Inventory and Vault contents. For Destiny 2, this scope is needed to read anything regarded as private. This is the only scope a Destiny 2 app needs for read operations against Destiny 2 data such as inventory, vault, currency, vendors, milestones, progression, etc. |
| `ReadUserData` | 128 | Read user data such as who they are web notifications, clan/group memberships, recent activity, muted users. |
| `EditUserData` | 256 | Edit user data such as preferred language, status, motto, avatar selection and theme. |
| `ReadDestinyVendorsAndAdvisors` | 512 | Access vendor and advisor data specific to a user. OBSOLETE. This scope is only used on the Destiny 1 API. |
| `ReadAndApplyTokens` | 1024 | Read offer history and claim and apply tokens for the user. |
| `AdvancedWriteActions` | 2048 | Can perform actions that will result in a prompt to the user via the Destiny app. |
| `PartnerOfferGrant` | 4096 | Can use the partner offer api to claim rewards defined for a partner |
| `DestinyUnlockValueQuery` | 8192 | Allows an app to query sensitive information like unlock flags and values not available through normal methods. |
| `UserPiiRead` | 16384 | Allows an app to query sensitive user PII, most notably email information. |

#### Applications.ApiUsage

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `apiCalls` | array&lt;Applications.Series&gt; | Counts for on API calls made for the time range. |
| `throttledRequests` | array&lt;Applications.Series&gt; | Instances of blocked requests or requests that crossed the warn threshold during the time range. |

#### Applications.Series

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `datapoints` | array&lt;Applications.Datapoint&gt; | Collection of samples with time and value. |
| `target` | string | Target to which to datapoints apply. |

#### Applications.Datapoint

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `time` | date-time | Timestamp for the related count. |
| `count` | double? | Count associated with timestamp |

#### Applications.Application

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `applicationType` | int32 | — |
| `applicationId` | int32 | Unique ID assigned to the application |
| `name` | string | Name of the application |
| `redirectUrl` | string | URL used to pass the user's authorization code to the application |
| `link` | string | Link to website for the application where a user can learn more about the app. |
| `scope` | int64 | Permissions the application needs to work |
| `origin` | string | Value of the Origin header sent in requests generated by this application. |
| `status` | int32 | Current status of the application. |
| `creationDate` | date-time | Date the application was first added to our database. |
| `statusChanged` | date-time | Date the application status last changed. |
| `firstPublished` | date-time | Date the first time the application status entered the 'Public' status. |
| `team` | array&lt;Applications.ApplicationDeveloper&gt; | List of team members who manage this application on Bungie.net. Will always consist of at least the application owner. |
| `overrideAuthorizeViewName` | string | An optional override for the Authorize view name. |

#### Applications.OAuthApplicationTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Confidential` | 1 | Indicates the application is server based and can keep its secrets from end users and other potential snoops. |
| `Public` | 2 | Indicates the application runs in a public place, and it can't be trusted to keep a secret. |

#### Applications.ApplicationStatusEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | No value assigned |
| `Private` | 1 | Application exists and works but will not appear in any public catalog. New applications start in this state, test applications will remain in this state. |
| `Public` | 2 | Active applications that can appear in an catalog. |
| `Disabled` | 3 | Application disabled by the owner. All authorizations will be treated as terminated while in this state. Owner can move back to private or public state. |
| `Blocked` | 4 | Application has been blocked by Bungie. It cannot be transitioned out of this state by the owner. Authorizations are terminated when an application is in this state. |

#### Applications.ApplicationDeveloper

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `role` | int32 | — |
| `apiEulaVersion` | int32 | — |
| `user` | User.UserInfoCard | — |

#### Applications.DeveloperRoleEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Owner` | 1 | — |
| `TeamMember` | 2 | — |

### Namespace: Common

#### Common.Models.CoreSettingsConfiguration

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `environment` | string | — |
| `systems` | Mapping&lt;string, Common.Models.CoreSystem&gt; | — |
| `ignoreReasons` | array&lt;Common.Models.CoreSetting&gt; | — |
| `forumCategories` | array&lt;Common.Models.CoreSetting&gt; | — |
| `groupAvatars` | array&lt;Common.Models.CoreSetting&gt; | — |
| `defaultGroupTheme` | Common.Models.CoreSetting | — |
| `destinyMembershipTypes` | array&lt;Common.Models.CoreSetting&gt; | — |
| `recruitmentPlatformTags` | array&lt;Common.Models.CoreSetting&gt; | — |
| `recruitmentMiscTags` | array&lt;Common.Models.CoreSetting&gt; | — |
| `recruitmentActivities` | array&lt;Common.Models.CoreSetting&gt; | — |
| `userContentLocales` | array&lt;Common.Models.CoreSetting&gt; | — |
| `systemContentLocales` | array&lt;Common.Models.CoreSetting&gt; | — |
| `clanBannerDecals` | array&lt;Common.Models.CoreSetting&gt; | — |
| `clanBannerDecalColors` | array&lt;Common.Models.CoreSetting&gt; | — |
| `clanBannerGonfalons` | array&lt;Common.Models.CoreSetting&gt; | — |
| `clanBannerGonfalonColors` | array&lt;Common.Models.CoreSetting&gt; | — |
| `clanBannerGonfalonDetails` | array&lt;Common.Models.CoreSetting&gt; | — |
| `clanBannerGonfalonDetailColors` | array&lt;Common.Models.CoreSetting&gt; | — |
| `clanBannerStandards` | array&lt;Common.Models.CoreSetting&gt; | — |
| `destiny2CoreSettings` | Common.Models.Destiny2CoreSettings | — |
| `emailSettings` | User.EmailSettings | — |
| `fireteamActivities` | array&lt;Common.Models.CoreSetting&gt; | — |

#### Common.Models.CoreSystem

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `enabled` | boolean | — |
| `parameters` | Mapping&lt;string, string&gt; | — |

#### Common.Models.CoreSetting

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `identifier` | string | — |
| `isDefault` | boolean | — |
| `displayName` | string | — |
| `summary` | string | — |
| `imagePath` | string | — |
| `childSettings` | array&lt;Common.Models.CoreSetting&gt; | — |

#### Common.Models.Destiny2CoreSettings

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `collectionRootNode` | uint32 → DestinyPresentationNodeDefinition | — |
| `badgesRootNode` | uint32 → DestinyPresentationNodeDefinition | — |
| `recordsRootNode` | uint32 → DestinyPresentationNodeDefinition | — |
| `medalsRootNode` | uint32 → DestinyPresentationNodeDefinition | — |
| `metricsRootNode` | uint32 → DestinyPresentationNodeDefinition | — |
| `activeTriumphsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `activeSealsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `legacyTriumphsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `legacySealsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `medalsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `exoticCatalystsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `loreRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `craftingRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `globalConstantsHash` | uint32 → DestinyGlobalConstantsDefinition | — |
| `loadoutConstantsHash` | uint32 → DestinyLoadoutConstantsDefinition | — |
| `guardianRankConstantsHash` | uint32 → DestinyGuardianRankConstantsDefinition | — |
| `fireteamFinderConstantsHash` | uint32 → DestinyFireteamFinderConstantsDefinition | — |
| `inventoryItemConstantsHash` | uint32 → DestinyInventoryItemConstantsDefinition | — |
| `featuredItemsListHash` | uint32 → DestinyItemFilterDefinition | — |
| `armorArchetypePlugSetHash` | uint32 → DestinyPlugSetDefinition | — |
| `seasonalHubEventCardHash` | uint32 → DestinyEventCardDefinition | — |
| `guardianRanksRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `currentRankProgressionHashes` | array&lt;uint32&gt; → DestinyProgressionDefinition | — |
| `insertPlugFreeProtectedPlugItemHashes` | array&lt;uint32&gt; → DestinyInventoryItemDefinition | — |
| `insertPlugFreeBlockedSocketTypeHashes` | array&lt;uint32&gt; → DestinySocketTypeDefinition | — |
| `enabledFireteamFinderActivityGraphHashes` | array&lt;uint32&gt; → DestinyFireteamFinderActivityGraphDefinition | — |
| `undiscoveredCollectibleImage` | string | — |
| `ammoTypeHeavyIcon` | string | — |
| `ammoTypeSpecialIcon` | string | — |
| `ammoTypePrimaryIcon` | string | — |
| `currentSeasonalArtifactHash` | uint32 → DestinyVendorDefinition | — |
| `currentSeasonHash` | uint32 → DestinySeasonDefinition? | — |
| `currentSeasonPassHash` | uint32 → DestinySeasonPassDefinition? | — |
| `seasonalChallengesPresentationNodeHash` | uint32 → DestinyPresentationNodeDefinition? | — |
| `futureSeasonHashes` | array&lt;uint32&gt; → DestinySeasonDefinition | — |
| `pastSeasonHashes` | array&lt;uint32&gt; → DestinySeasonDefinition | — |

### Namespace: Components

#### Components.ComponentResponse

**Type:** object

The base class for any component-returning object that may need to indicate information about the state of the component being returned.

| Property | Type | Description |
| --- | --- | --- |
| `privacy` | int32 | — |
| `disabled` | boolean? | If true, this component is disabled. |

#### Components.ComponentPrivacySettingEnumeration

**Enum** (`int32`)

A set of flags for reason(s) why the component populated in the way that it did. Inspect the individual flags for the reasons.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Public` | 1 | — |
| `Private` | 2 | — |

### Namespace: Config

#### Config.UserTheme

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `userThemeId` | int32 | — |
| `userThemeName` | string | — |
| `userThemeDescription` | string | — |

#### Config.GroupTheme

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `name` | string | — |
| `folder` | string | — |
| `description` | string | — |

#### Config.ClanBanner.ClanBannerSource

**Type:** object

(Empty object — no properties.)

#### Config.ClanBanner.ClanBannerDecal

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `identifier` | string | — |
| `foregroundPath` | string | — |
| `backgroundPath` | string | — |

### Namespace: Content

#### Content.Models.ContentTypeDescription

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `cType` | string | — |
| `name` | string | — |
| `contentDescription` | string | — |
| `previewImage` | string | — |
| `priority` | int32 | — |
| `reminder` | string | — |
| `properties` | array&lt;Content.Models.ContentTypeProperty&gt; | — |
| `tagMetadata` | array&lt;Content.Models.TagMetadataDefinition&gt; | — |
| `tagMetadataItems` | Mapping&lt;string, Content.Models.TagMetadataItem&gt; | — |
| `usageExamples` | array&lt;string&gt; | — |
| `showInContentEditor` | boolean | — |
| `typeOf` | string | — |
| `bindIdentifierToProperty` | string | — |
| `boundRegex` | string | — |
| `forceIdentifierBinding` | boolean | — |
| `allowComments` | boolean | — |
| `autoEnglishPropertyFallback` | boolean | — |
| `bulkUploadable` | boolean | — |
| `previews` | array&lt;Content.Models.ContentPreview&gt; | — |
| `suppressCmsPath` | boolean | — |
| `propertySections` | array&lt;Content.Models.ContentTypePropertySection&gt; | — |

#### Content.Models.ContentTypeProperty

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `name` | string | — |
| `rootPropertyName` | string | — |
| `readableName` | string | — |
| `value` | string | — |
| `propertyDescription` | string | — |
| `localizable` | boolean | — |
| `fallback` | boolean | — |
| `enabled` | boolean | — |
| `order` | int32 | — |
| `visible` | boolean | — |
| `isTitle` | boolean | — |
| `required` | boolean | — |
| `maxLength` | int32 | — |
| `maxByteLength` | int32 | — |
| `maxFileSize` | int32 | — |
| `regexp` | string | — |
| `validateAs` | string | — |
| `rssAttribute` | string | — |
| `visibleDependency` | string | — |
| `visibleOn` | string | — |
| `datatype` | int32 | — |
| `attributes` | Mapping&lt;string, string&gt; | — |
| `childProperties` | array&lt;Content.Models.ContentTypeProperty&gt; | — |
| `contentTypeAllowed` | string | — |
| `bindToProperty` | string | — |
| `boundRegex` | string | — |
| `representationSelection` | Mapping&lt;string, string&gt; | — |
| `defaultValues` | array&lt;Content.Models.ContentTypeDefaultValue&gt; | — |
| `isExternalAllowed` | boolean | — |
| `propertySection` | string | — |
| `weight` | int32 | — |
| `entitytype` | string | — |
| `isCombo` | boolean | — |
| `suppressProperty` | boolean | — |
| `legalContentTypes` | array&lt;string&gt; | — |
| `representationValidationString` | string | — |
| `minWidth` | int32 | — |
| `maxWidth` | int32 | — |
| `minHeight` | int32 | — |
| `maxHeight` | int32 | — |
| `isVideo` | boolean | — |
| `isImage` | boolean | — |

#### Content.Models.ContentPropertyDataTypeEnumEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Plaintext` | 1 | — |
| `Html` | 2 | — |
| `Dropdown` | 3 | — |
| `List` | 4 | — |
| `Json` | 5 | — |
| `Content` | 6 | — |
| `Representation` | 7 | — |
| `Set` | 8 | — |
| `File` | 9 | — |
| `FolderSet` | 10 | — |
| `Date` | 11 | — |
| `MultilinePlaintext` | 12 | — |
| `DestinyContent` | 13 | — |
| `Color` | 14 | — |

#### Content.Models.ContentTypeDefaultValue

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `whenClause` | string | — |
| `whenValue` | string | — |
| `defaultValue` | string | — |

#### Content.Models.TagMetadataDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `description` | string | — |
| `order` | int32 | — |
| `items` | array&lt;Content.Models.TagMetadataItem&gt; | — |
| `datatype` | string | — |
| `name` | string | — |
| `isRequired` | boolean | — |

#### Content.Models.TagMetadataItem

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `description` | string | — |
| `tagText` | string | — |
| `groups` | array&lt;string&gt; | — |
| `isDefault` | boolean | — |
| `name` | string | — |

#### Content.Models.ContentPreview

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `name` | string | — |
| `path` | string | — |
| `itemInSet` | boolean | — |
| `setTag` | string | — |
| `setNesting` | int32 | — |
| `useSetId` | int32 | — |

#### Content.Models.ContentTypePropertySection

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `name` | string | — |
| `readableName` | string | — |
| `collapsed` | boolean | — |

#### Content.ContentItemPublicContract

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `contentId` | int64 | — |
| `cType` | string | — |
| `cmsPath` | string | — |
| `creationDate` | date-time | — |
| `modifyDate` | date-time | — |
| `allowComments` | boolean | — |
| `hasAgeGate` | boolean | — |
| `minimumAge` | int32 | — |
| `ratingImagePath` | string | — |
| `author` | User.GeneralUser | — |
| `autoEnglishPropertyFallback` | boolean | — |
| `properties` | Mapping&lt;string, object&gt; | Firehose content is really a collection of metadata and "properties", which are the potentially-but- not-strictly localizable data that comprises the meat of whatever content is being shown. As Cole Porter would have crooned, "Anything Goes" with Firehose properties. They are most often strings, but they can theoretically be anything. They are JSON encoded, and could be JSON structures, simple strings, numbers etc... The Content Type of the item (cType) will describe the properties, and thus how they ought to be deserialized. |
| `representations` | array&lt;Content.ContentRepresentation&gt; | — |
| `tags` | array&lt;string&gt; | NOTE: Tags will always be lower case. |
| `commentSummary` | Content.CommentSummary | — |

#### Content.ContentRepresentation

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `name` | string | — |
| `path` | string | — |
| `validationString` | string | — |

#### Content.CommentSummary

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `topicId` | int64 | — |
| `commentCount` | int32 | — |

#### Content.NewsArticleRssResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `CurrentPaginationToken` | int32 | — |
| `NextPaginationToken` | int32? | — |
| `ResultCountThisPage` | int32 | — |
| `NewsArticles` | array&lt;Content.NewsArticleRssItem&gt; | — |
| `CategoryFilter` | string | — |
| `PagerAction` | string | — |

#### Content.NewsArticleRssItem

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `Title` | string | — |
| `Link` | string | — |
| `PubDate` | date-time | — |
| `UniqueIdentifier` | string | — |
| `Description` | string | — |
| `HtmlContent` | string | — |
| `ImagePath` | string | — |
| `OptionalMobileImagePath` | string | — |

### Namespace: Dates

#### Dates.DateRange

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `start` | date-time | — |
| `end` | date-time | — |

### Namespace: Destiny

#### Destiny.DestinyProgression

**Type:** object

Information about a current character's status with a Progression. A progression is a value that can increase with activity and has levels. Think Character Level and Reputation Levels. Combine this "live" data with the related DestinyProgressionDefinition for a full picture of the Progression.

| Property | Type | Description |
| --- | --- | --- |
| `progressionHash` | uint32 → DestinyProgressionDefinition | The hash identifier of the Progression in question. Use it to look up the DestinyProgressionDefinition in static data. |
| `dailyProgress` | int32 | The amount of progress earned today for this progression. |
| `dailyLimit` | int32 | If this progression has a daily limit, this is that limit. |
| `weeklyProgress` | int32 | The amount of progress earned toward this progression in the current week. |
| `weeklyLimit` | int32 | If this progression has a weekly limit, this is that limit. |
| `currentProgress` | int32 | This is the total amount of progress obtained overall for this progression (for instance, the total amount of Character Level experience earned) |
| `level` | int32 | This is the level of the progression (for instance, the Character Level). |
| `levelCap` | int32 | This is the maximum possible level you can achieve for this progression (for example, the maximum character level obtainable) |
| `stepIndex` | int32 | Progressions define their levels in "steps". Since the last step may be repeatable, the user may be at a higher level than the actual Step achieved in the progression. Not necessarily useful, but potentially interesting for those cruising the API. Relate this to the "steps" property of the DestinyProgression to see which step the user is on, if you care about that. (Note that this is Content Version dependent since it refers to indexes.) |
| `progressToNextLevel` | int32 | The amount of progression (i.e. "Experience") needed to reach the next level of this Progression. Jeez, progression is such an overloaded word. |
| `nextLevelAt` | int32 | The total amount of progression (i.e. "Experience") needed in order to reach the next level. |
| `currentResetCount` | int32? | The number of resets of this progression you've executed this season, if applicable to this progression. |
| `seasonResets` | array&lt;Destiny.DestinyProgressionResetEntry&gt; | Information about historical resets of this progression, if there is any data for it. |
| `rewardItemStates` | array&lt;int32&gt; | Information about historical rewards for this progression, if there is any data for it. |
| `rewardItemSocketOverrideStates` | Mapping&lt;int32, Destiny.DestinyProgressionRewardItemSocketOverrideState&gt; | Information about items stats and states that have socket overrides, if there is any data for it. |

#### Destiny.DestinyProgressionResetEntry

**Type:** object

Represents a season and the number of resets you had in that season. We do not necessarily - even for progressions with resets - track it over all seasons. So be careful and check the season numbers being returned.

| Property | Type | Description |
| --- | --- | --- |
| `season` | int32 | — |
| `resets` | int32 | — |

#### Destiny.DestinyProgressionRewardItemStateEnumeration

**Enum** (`int32`)

Represents the different states a progression reward item can be in.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Invisible` | 1 | If this is set, the reward should be hidden. |
| `Earned` | 2 | If this is set, the reward has been earned. |
| `Claimed` | 4 | If this is set, the reward has been claimed. |
| `ClaimAllowed` | 8 | If this is set, the reward is allowed to be claimed by this Character. An item can be earned but still can't be claimed in certain circumstances, like if it's only allowed for certain subclasses. It also might not be able to be claimed if you already claimed it! |

#### Destiny.DestinyProgressionRewardItemSocketOverrideState

**Type:** object

Represents the stats and item state if applicable for progression reward items with socket overrides

| Property | Type | Description |
| --- | --- | --- |
| `rewardItemStats` | Mapping&lt;uint32, Destiny.DestinyStat&gt; → DestinyStatDefinition | Information about the computed stats from socket and plug overrides for this progression, if there is any data for it. |
| `itemState` | int32 | Information about the item state, specifically deepsight if there is any data for it |

#### Destiny.DestinyStat

**Type:** object

Represents a stat on an item *or* Character (NOT a Historical Stat, but a physical attribute stat like Attack, Defense etc...)

| Property | Type | Description |
| --- | --- | --- |
| `statHash` | uint32 → DestinyStatDefinition | The hash identifier for the Stat. Use it to look up the DestinyStatDefinition for static data about the stat. |
| `value` | int32 | The current value of the Stat. |

#### Destiny.Definitions.DestinyDefinition

**Type:** object

Provides common properties for destiny definitions.

| Property | Type | Description |
| --- | --- | --- |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyStatDefinition

**Object** · *(Manifest definition, table `Stats`)*

This represents a stat that's applied to a character or an item (such as a weapon, piece of armor, or a vehicle). An example of a stat might be Attack Power on a weapon. Stats go through a complex set of transformations before they end up being shown to the user as a number or a progress bar, and those transformations are fundamentally intertwined with the concept of a "Stat Group" (DestinyStatGroupDefinition). Items have both Stats and a reference to a Stat Group, and it is the Stat Group that takes the raw stat information and gives it both rendering metadata (such as whether to show it as a number or a progress bar) and the final transformation data (interpolation tables to turn the raw investment stat into a display stat). Please see DestinyStatGroupDefinition for more information on that transformational process. Stats are segregated from Stat Groups because different items and types of items can refer to the same stat, but have different "scales" for the stat while still having the same underlying value. For example, both a Shotgun and an Auto Rifle may have a "raw" impact stat of 50, but the Auto Rifle's Stat Group will scale that 50 down so that, when it is displayed, it is a smaller value relative to the shotgun. (this is a totally made up example, don't assume shotguns have naturally higher impact than auto rifles because of this) A final caveat is that some stats, even after this "final" transformation, go through yet another set of transformations directly in the game as a result of dynamic, stateful scripts that get run. BNet has no access to these scripts, nor any way to know which scripts get executed. As a result, the stats for an item that you see in-game - particularly for stats that are often impacted by Perks, like Magazine Size - can change dramatically from what we return on Bungie.Net. This is a known issue with no fix coming down the pipeline. Take these stats with a grain of salt. Stats actually go through four transformations, for those interested: 1) "Sandbox" stat, the "most raw" form. These are pretty much useless without transformations applied, and thus are not currently returned in the API. If you really want these, we can provide them. Maybe someone could do something cool with it? 2) "Investment" stat (the stat's value after DestinyStatDefinition's interpolation tables and aggregation logic is applied to the "Sandbox" stat value) 3) "Display" stat (the stat's base UI-visible value after DestinyStatGroupDefinition's interpolation tables are applied to the Investment Stat value. For most stats, this is what is displayed.) 4) Underlying in-game stat (the stat's actual value according to the game, after the game runs dynamic scripts based on the game and character's state. This is the final transformation that BNet does not have access to. For most stats, this is not actually displayed to the user, with the exception of Magazine Size which is then piped back to the UI for display in-game, but not to BNet.)

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `aggregationType` | int32 | Stats can exist on a character or an item, and they may potentially be aggregated in different ways. The DestinyStatAggregationType enum value indicates the way that this stat is being aggregated. |
| `hasComputedBlock` | boolean | True if the stat is computed rather than being delivered as a raw value on items. For instance, the Light stat in Destiny 1 was a computed stat. |
| `statCategory` | int32 | The category of the stat, according to the game. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition

**Type:** object

Many Destiny*Definition contracts - the "first order" entities of Destiny that have their own tables in the Manifest Database - also have displayable information. This is the base class for that display information.

| Property | Type | Description |
| --- | --- | --- |
| `description` | string | — |
| `name` | string | — |
| `icon` | string | Note that "icon" is sometimes misleading, and should be interpreted in the context of the entity. For instance, in Destiny 1 the DestinyRecordBookDefinition's icon was a big picture of a book. But usually, it will be a small square image that you can use as... well, an icon. They are currently represented as 96px x 96px images. |
| `iconHash` | uint32 → DestinyIconDefinition | — |
| `iconSequences` | array&lt;Destiny.Definitions.Common.DestinyIconSequenceDefinition&gt; | — |
| `highResIcon` | string | If this item has a high-res icon (at least for now, many things won't), then the path to that icon will be here. |
| `hasIcon` | boolean | — |

#### Destiny.Definitions.Common.DestinyIconSequenceDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `frames` | array&lt;string&gt; | — |

#### Destiny.Definitions.Inventory.DestinyIconDefinition

**Object** · *(Manifest definition, table `icons`)*

Lists of icons that can be used for a variety of purposes

| Property | Type | Description |
| --- | --- | --- |
| `foreground` | string | — |
| `background` | string | — |
| `secondaryBackground` | string | — |
| `specialBackground` | string | — |
| `highResForeground` | string | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.DestinyStatAggregationTypeEnumeration

**Enum** (`int32`)

When a Stat (DestinyStatDefinition) is aggregated, this is the rules used for determining the level and formula used for aggregation. \* CharacterAverage = apply a weighted average using the related DestinyStatGroupDefinition on the DestinyInventoryItemDefinition across the character's equipped items. See both of those definitions for details. \* Character = don't aggregate: the stat should be located and used directly on the character. \* Item = don't aggregate: the stat should be located and used directly on the item.

| Value | # | Description |
| --- | --- | --- |
| `CharacterAverage` | 0 | — |
| `Character` | 1 | — |
| `Item` | 2 | — |

#### Destiny.DestinyStatCategoryEnumeration

**Enum** (`int32`)

At last, stats have categories. Use this for whatever purpose you might wish.

| Value | # | Description |
| --- | --- | --- |
| `Gameplay` | 0 | — |
| `Weapon` | 1 | — |
| `Defense` | 2 | — |
| `Primary` | 3 | — |

#### Destiny.ItemStateEnumeration

**Enum** (`int32`)

A flags enumeration/bitmask where each bit represents a different possible state that the item can be in that may effect how the item is displayed to the user and what actions can be performed against it.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Locked` | 1 | If this bit is set, the item has been "locked" by the user and cannot be deleted. You may want to represent this visually with a "lock" icon. |
| `Tracked` | 2 | If this bit is set, the item is a quest that's being tracked by the user. You may want a visual indicator to show that this is a tracked quest. |
| `Masterwork` | 4 | If this bit is set, the item has a Masterwork plug inserted. This usually coincides with having a special "glowing" effect applied to the item's icon. |
| `Crafted` | 8 | If this bit is set, the item has been 'crafted' by the player. You may want to represent this visually with a "crafted" icon overlay. |
| `HighlightedObjective` | 16 | If this bit is set, the item has a 'highlighted' objective. You may want to represent this with an orange-red icon border color. |
| `Enhanced` | 32 | If this bit is set, the item has been 'enhanced' by the player. |

#### Destiny.Definitions.DestinyProgressionDefinition

**Object** · *(Manifest definition, table `Progressions`)*

A "Progression" in Destiny is best explained by an example. A Character's "Level" is a progression: it has Experience that can be earned, levels that can be gained, and is evaluated and displayed at various points in the game. A Character's "Faction Reputation" is also a progression for much the same reason. Progression is used by a variety of systems, and the definition of a Progression will generally only be useful if combining with live data (such as a character's DestinyCharacterProgressionComponent.progressions property, which holds that character's live Progression states). Fundamentally, a Progression measures your "Level" by evaluating the thresholds in its Steps (one step per level, except for the last step which can be repeated indefinitely for "Levels" that have no ceiling) against the total earned "progression points"/experience. (for simplicity purposes, we will henceforth refer to earned progression points as experience, though it need not be a mechanic that in any way resembles Experience in a traditional sense). Earned experience is calculated in a variety of ways, determined by the Progression's scope. These go from looking up a stored value to performing exceedingly obtuse calculations. This is why we provide live data in DestinyCharacterProgressionComponent.progressions, so you don't have to worry about those.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.DestinyProgressionDisplayPropertiesDefinition | — |
| `scope` | int32 | The "Scope" of the progression indicates the source of the progression's live data. See the DestinyProgressionScope enum for more info: but essentially, a Progression can either be backed by a stored value, or it can be a calculated derivative of other values. |
| `repeatLastStep` | boolean | If this is True, then the progression doesn't have a maximum level. |
| `source` | string | If there's a description of how to earn this progression in the local config, this will be that localized description. |
| `steps` | array&lt;Destiny.Definitions.DestinyProgressionStepDefinition&gt; | Progressions are divided into Steps, which roughly equate to "Levels" in the traditional sense of a Progression. Notably, the last step can be repeated indefinitely if repeatLastStep is true, meaning that the calculation for your level is not as simple as comparing your current progress to the max progress of the steps. These and more calculations are done for you if you grab live character progression data, such as in the DestinyCharacterProgressionComponent. |
| `visible` | boolean | If true, the Progression is something worth showing to users. If false, BNet isn't going to show it. But that doesn't mean you can't. We're all friends here. |
| `factionHash` | uint32 → DestinyFactionDefinition? | If the value exists, this is the hash identifier for the Faction that owns this Progression. This is purely for convenience, if you're looking at a progression and want to know if and who it's related to in terms of Faction Reputation. |
| `color` | Destiny.Misc.DestinyColor | The #RGB string value for the color related to this progression, if there is one. |
| `rankIcon` | string | For progressions that have it, this is the rank icon we use in the Companion, displayed above the progressions' rank value. |
| `rewardItems` | array&lt;Destiny.Definitions.DestinyProgressionRewardItemQuantity&gt; | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyProgressionDisplayPropertiesDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `displayUnitsName` | string | When progressions show your "experience" gained, that bar has units (i.e. "Experience", "Bad Dudes Snuffed Out", whatever). This is the localized string for that unit of measurement. |
| `description` | string | — |
| `name` | string | — |
| `icon` | string | Note that "icon" is sometimes misleading, and should be interpreted in the context of the entity. For instance, in Destiny 1 the DestinyRecordBookDefinition's icon was a big picture of a book. But usually, it will be a small square image that you can use as... well, an icon. They are currently represented as 96px x 96px images. |
| `iconHash` | uint32 → DestinyIconDefinition | — |
| `iconSequences` | array&lt;Destiny.Definitions.Common.DestinyIconSequenceDefinition&gt; | — |
| `highResIcon` | string | If this item has a high-res icon (at least for now, many things won't), then the path to that icon will be here. |
| `hasIcon` | boolean | — |

#### Destiny.DestinyProgressionScopeEnumeration

**Enum** (`int32`)

There are many Progressions in Destiny (think Character Level, or Reputation). These are the various "Scopes" of Progressions, which affect many things: \* Where/if they are stored \* How they are calculated \* Where they can be used in other game logic

| Value | # | Description |
| --- | --- | --- |
| `Account` | 0 | — |
| `Character` | 1 | — |
| `Clan` | 2 | — |
| `Item` | 3 | — |
| `ImplicitFromEquipment` | 4 | — |
| `Mapped` | 5 | — |
| `MappedAggregate` | 6 | — |
| `MappedStat` | 7 | — |
| `MappedUnlockValue` | 8 | — |

#### Destiny.Definitions.DestinyProgressionStepDefinition

**Type:** object

This defines a single Step in a progression (which roughly equates to a level. See DestinyProgressionDefinition for caveats).

| Property | Type | Description |
| --- | --- | --- |
| `stepName` | string | Very rarely, Progressions will have localized text describing the Level of the progression. This will be that localized text, if it exists. Otherwise, the standard appears to be to simply show the level numerically. |
| `displayEffectType` | int32 | This appears to be, when you "level up", whether a visual effect will display and on what entity. See DestinyProgressionStepDisplayEffect for slightly more info. |
| `progressTotal` | int32 | The total amount of progression points/"experience" you will need to initially reach this step. If this is the last step and the progression is repeating indefinitely (DestinyProgressionDefinition.repeatLastStep), this will also be the progress needed to level it up further by repeating this step again. |
| `rewardItems` | array&lt;Destiny.DestinyItemQuantity&gt; | A listing of items rewarded as a result of reaching this level. |
| `icon` | string | If this progression step has a specific icon related to it, this is the icon to show. |

#### Destiny.DestinyProgressionStepDisplayEffectEnumeration

**Enum** (`int32`)

If progression is earned, this determines whether the progression shows visual effects on the character or its item - or neither.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Character` | 1 | — |
| `Item` | 2 | — |

#### Destiny.DestinyItemQuantity

**Type:** object

Used in a number of Destiny contracts to return data about an item stack and its quantity. Can optionally return an itemInstanceId if the item is instanced - in which case, the quantity returned will be 1. If it's not... uh, let me know okay? Thanks.

| Property | Type | Description |
| --- | --- | --- |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier for the item in question. Use it to look up the item's DestinyInventoryItemDefinition. |
| `itemInstanceId` | int64? | If this quantity is referring to a specific instance of an item, this will have the item's instance ID. Normally, this will be null. |
| `quantity` | int32 | The amount of the item needed/available depending on the context of where DestinyItemQuantity is being used. |
| `hasConditionalVisibility` | boolean | Indicates that this item quantity may be conditionally shown or hidden, based on various sources of state. For example: server flags, account state, or character progress. |

#### Destiny.Definitions.DestinyInventoryItemDefinition

**Object** · *(Manifest definition, table `Items`)*

So much of what you see in Destiny is actually an Item used in a new and creative way. This is the definition for Items in Destiny, which started off as just entities that could exist in your Inventory but ended up being the backing data for so much more: quests, reward previews, slots, and subclasses. In practice, you will want to associate this data with "live" item data from a Bungie.Net Platform call: these definitions describe the item in generic, non-instanced terms: but an actual instance of an item can vary widely from these generic definitions.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `tooltipNotifications` | array&lt;Destiny.Definitions.DestinyItemTooltipNotification&gt; | Tooltips that only come up conditionally for the item. Check the live data DestinyItemComponent.tooltipNotificationIndexes property for which of these should be shown at runtime. |
| `collectibleHash` | uint32 → DestinyCollectibleDefinition? | If this item has a collectible related to it, this is the hash identifier of that collectible entry. |
| `iconWatermark` | string | If available, this is the original 'active' release watermark overlay for the icon. If the item has different versions, this can be overridden by the 'display version watermark icon' from the 'quality' block. Alternatively, if there is no watermark for the version, and the item version has a power cap below the current season power cap, this can be overridden by the iconWatermarkShelved property. |
| `iconWatermarkShelved` | string | If available, this is the 'shelved' release watermark overlay for the icon. If the item version has a power cap below the current season power cap, it can be treated as 'shelved', and should be shown with this 'shelved' watermark overlay. |
| `iconWatermarkFeatured` | string | This is the active watermark for the item if it is currently Featured in-game. Clients should use the isFeaturedItem boolean to decide whether or not to show this as opposed to iconWatermark. |
| `secondaryIcon` | string | A secondary icon associated with the item. Currently this is used in very context specific applications, such as Emblem Nameplates. |
| `secondaryOverlay` | string | Pulled from the secondary icon, this is the "secondary background" of the secondary icon. Confusing? Sure, that's why I call it "overlay" here: because as far as it's been used thus far, it has been for an optional overlay image. We'll see if that holds up, but at least for now it explains what this image is a bit better. |
| `secondarySpecial` | string | Pulled from the Secondary Icon, this is the "special" background for the item. For Emblems, this is the background image used on the Details view: but it need not be limited to that for other types of items. |
| `backgroundColor` | Destiny.Misc.DestinyColor | Sometimes, an item will have a background color. Most notably this occurs with Emblems, who use the Background Color for small character nameplates such as the "friends" view you see in- game. There are almost certainly other items that have background color as well, though I have not bothered to investigate what items have it nor what purposes they serve: use it as you will. |
| `isFeaturedItem` | boolean | Whether or not this item is currently featured in the game, giving it a special watermark |
| `isHolofoil` | boolean | Whether or not this item is holofoil, which has special icon treatment and in-game appearance. |
| `isAdept` | boolean | Whether or not this item is adept, which has increased stats and/or perks. |
| `screenshot` | string | If we were able to acquire an in-game screenshot for the item, the path to that screenshot will be returned here. Note that not all items have screenshots: particularly not any non-equippable items. |
| `itemTypeDisplayName` | string | The localized title/name of the item's type. This can be whatever the designers want, and has no guarantee of consistency between items. |
| `flavorText` | string | — |
| `uiItemDisplayStyle` | string | A string identifier that the game's UI uses to determine how the item should be rendered in inventory screens and the like. This could really be anything - at the moment, we don't have the time to really breakdown and maintain all the possible strings this could be, partly because new ones could be added ad hoc. But if you want to use it to dictate your own UI, or look for items with a certain display style, go for it! |
| `itemTypeAndTierDisplayName` | string | It became a common enough pattern in our UI to show Item Type and Tier combined into a single localized string that I'm just going to go ahead and start pre-creating these for items. |
| `displaySource` | string | In theory, it is a localized string telling you about how you can find the item. I really wish this was more consistent. Many times, it has nothing. Sometimes, it's instead a more narrative-forward description of the item. Which is cool, and I wish all properties had that data, but it should really be its own property. |
| `tooltipStyle` | string | An identifier that the game UI uses to determine what type of tooltip to show for the item. These have no corresponding definitions that BNet can link to: so it'll be up to you to interpret and display your UI differently according to these styles (or ignore it). |
| `action` | Destiny.Definitions.DestinyItemActionBlockDefinition | If the item can be "used", this block will be non-null, and will have data related to the action performed when using the item. (Guess what? 99% of the time, this action is "dismantle". Shocker) |
| `crafting` | Destiny.Definitions.DestinyItemCraftingBlockDefinition | Recipe items will have relevant crafting information available here. |
| `inventory` | Destiny.Definitions.DestinyItemInventoryBlockDefinition | If this item can exist in an inventory, this block will be non-null. In practice, every item that currently exists has one of these blocks. But note that it is not necessarily guaranteed. |
| `setData` | Destiny.Definitions.DestinyItemSetBlockDefinition | If this item is a quest, this block will be non-null. In practice, I wish I had called this the Quest block, but at the time it wasn't clear to me whether it would end up being used for purposes other than quests. It will contain data about the steps in the quest, and mechanics we can use for displaying and tracking the quest. |
| `stats` | Destiny.Definitions.DestinyItemStatBlockDefinition | If this item can have stats (such as a weapon, armor, or vehicle), this block will be non-null and populated with the stats found on the item. |
| `emblemObjectiveHash` | uint32? | If the item is an emblem that has a special Objective attached to it - for instance, if the emblem tracks PVP Kills, or what-have-you. This is a bit different from, for example, the Vanguard Kill Tracker mod, which pipes data into the "art channel". When I get some time, I would like to standardize these so you can get at the values they expose without having to care about what they're being used for and how they are wired up, but for now here's the raw data. |
| `equippingBlock` | Destiny.Definitions.DestinyEquippingBlockDefinition | If this item can be equipped, this block will be non-null and will be populated with the conditions under which it can be equipped. |
| `translationBlock` | Destiny.Definitions.DestinyItemTranslationBlockDefinition | If this item can be rendered, this block will be non-null and will be populated with rendering information. |
| `preview` | Destiny.Definitions.DestinyItemPreviewBlockDefinition | If this item can be Used or Acquired to gain other items (for instance, how Eververse Boxes can be consumed to get items from the box), this block will be non-null and will give summary information for the items that can be acquired. |
| `quality` | Destiny.Definitions.DestinyItemQualityBlockDefinition | If this item can have a level or stats, this block will be non-null and will be populated with default quality (item level, "quality", and infusion) data. See the block for more details, there's often less upfront information in D2 so you'll want to be aware of how you use quality and item level on the definition level now. |
| `value` | Destiny.Definitions.DestinyItemValueBlockDefinition | The conceptual "Value" of an item, if any was defined. See the DestinyItemValueBlockDefinition for more details. |
| `sourceData` | Destiny.Definitions.DestinyItemSourceBlockDefinition | If this item has a known source, this block will be non-null and populated with source information. Unfortunately, at this time we are not generating sources: that is some aggressively manual work which we didn't have time for, and I'm hoping to get back to at some point in the future. |
| `objectives` | Destiny.Definitions.DestinyItemObjectiveBlockDefinition | If this item has Objectives (extra tasks that can be accomplished related to the item... most frequently when the item is a Quest Step and the Objectives need to be completed to move on to the next Quest Step), this block will be non-null and the objectives defined herein. |
| `metrics` | Destiny.Definitions.DestinyItemMetricBlockDefinition | If this item has available metrics to be shown, this block will be non-null have the appropriate hashes defined. |
| `plug` | Destiny.Definitions.Items.DestinyItemPlugDefinition | If this item *is* a Plug, this will be non-null and the info defined herein. See DestinyItemPlugDefinition for more information. |
| `gearset` | Destiny.Definitions.DestinyItemGearsetBlockDefinition | If this item has related items in a "Gear Set", this will be non-null and the relationships defined herein. |
| `sack` | Destiny.Definitions.DestinyItemSackBlockDefinition | If this item is a "reward sack" that can be opened to provide other items, this will be non-null and the properties of the sack contained herein. |
| `sockets` | Destiny.Definitions.DestinyItemSocketBlockDefinition | If this item has any Sockets, this will be non-null and the individual sockets on the item will be defined herein. |
| `summary` | Destiny.Definitions.DestinyItemSummaryBlockDefinition | Summary data about the item. |
| `talentGrid` | Destiny.Definitions.DestinyItemTalentGridBlockDefinition | If the item has a Talent Grid, this will be non-null and the properties of the grid defined herein. Note that, while many items still have talent grids, the only ones with meaningful Nodes still on them will be Subclass/"Build" items. |
| `investmentStats` | array&lt;Destiny.Definitions.DestinyItemInvestmentStatDefinition&gt; | If the item has stats, this block will be defined. It has the "raw" investment stats for the item. These investment stats don't take into account the ways that the items can spawn, nor do they take into account any Stat Group transformations. I have retained them for debugging purposes, but I do not know how useful people will find them. |
| `perks` | array&lt;Destiny.Definitions.DestinyItemPerkEntryDefinition&gt; | If the item has any *intrinsic* Perks (Perks that it will provide regardless of Sockets, Talent Grid, and other transitory state), they will be defined here. |
| `loreHash` | uint32 → DestinyLoreDefinition? | If the item has any related Lore (DestinyLoreDefinition), this will be the hash identifier you can use to look up the lore definition. |
| `summaryItemHash` | uint32 → DestinyInventoryItemDefinition? | There are times when the game will show you a "summary/vague" version of an item - such as a description of its type represented as a DestinyInventoryItemDefinition - rather than display the item itself. This happens sometimes when summarizing possible rewards in a tooltip. This is the item displayed instead, if it exists. |
| `animations` | array&lt;Destiny.Definitions.Animations.DestinyAnimationReference&gt; | If any animations were extracted from game content for this item, these will be the definitions of those animations. |
| `allowActions` | boolean | BNet may forbid the execution of actions on this item via the API. If that is occurring, allowActions will be set to false. |
| `links` | array&lt;Links.HyperlinkReference&gt; | If we added any help or informational URLs about this item, these will be those links. |
| `doesPostmasterPullHaveSideEffects` | boolean | The boolean will indicate to us (and you!) whether something *could* happen when you transfer this item from the Postmaster that might be considered a "destructive" action. It is not feasible currently to tell you (or ourelves!) in a consistent way whether this *will* actually cause a destructive action, so we are playing it safe: if it has the potential to do so, we will not allow it to be transferred from the Postmaster by default. You will need to check for this flag before transferring an item from the Postmaster, or else you'll end up receiving an error. |
| `nonTransferrable` | boolean | The intrinsic transferability of an item. I hate that this boolean is negative - but there's a reason. Just because an item is intrinsically transferrable doesn't mean that it can be transferred, and we don't want to imply that this is the only source of that transferability. |
| `itemCategoryHashes` | array&lt;uint32&gt; → DestinyItemCategoryDefinition | BNet attempts to make a more formal definition of item "Categories", as defined by DestinyItemCategoryDefinition. This is a list of all Categories that we were able to algorithmically determine that this item is a member of. (for instance, that it's a "Weapon", that it's an "Auto Rifle", etc...) The algorithm for these is, unfortunately, volatile. If you believe you see a miscategorized item, please let us know on the Bungie API forums. |
| `specialItemType` | int32 | In Destiny 1, we identified some items as having particular categories that we'd like to know about for various internal logic purposes. These are defined in SpecialItemType, and while these days the itemCategoryHashes are the preferred way of identifying types, we have retained this enum for its convenience. |
| `itemType` | int32 | A value indicating the "base" the of the item. This enum is a useful but dramatic oversimplification of what it means for an item to have a "Type". Still, it's handy in many situations. itemCategoryHashes are the preferred way of identifying types, we have retained this enum for its convenience. |
| `itemSubType` | int32 | A value indicating the "sub-type" of the item. For instance, where an item might have an itemType value "Weapon", this will be something more specific like "Auto Rifle". itemCategoryHashes are the preferred way of identifying types, we have retained this enum for its convenience. |
| `classType` | int32 | We run a similarly weak-sauce algorithm to try and determine whether an item is restricted to a specific class. If we find it to be restricted in such a way, we set this classType property to match the class' enumeration value so that users can easily identify class restricted items. If you see a mis-classed item, please inform the developers in the Bungie API forum. |
| `breakerType` | int32 | Some weapons and plugs can have a "Breaker Type": a special ability that works sort of like damage type vulnerabilities. This is (almost?) always set on items by plugs. |
| `breakerTypeHash` | uint32 → DestinyBreakerTypeDefinition? | Since we also have a breaker type definition, this is the hash for that breaker type for your convenience. Whether you use the enum or hash and look up the definition depends on what's cleanest for your code. |
| `equippable` | boolean | If true, then you will be allowed to equip the item if you pass its other requirements. This being false means that you cannot equip the item under any circumstances. |
| `damageTypeHashes` | array&lt;uint32&gt; → DestinyDamageTypeDefinition | Theoretically, an item can have many possible damage types. In *practice*, this is not true, but just in case weapons start being made that have multiple (for instance, an item where a socket has reusable plugs for every possible damage type that you can choose from freely), this field will return all of the possible damage types that are available to the weapon by default. |
| `damageTypes` | array&lt;int32&gt; | This is the list of all damage types that we know ahead of time the item can take on. Unfortunately, this does not preclude the possibility of something funky happening to give the item a damage type that cannot be predicted beforehand: for example, if some designer decides to create arbitrary non-reusable plugs that cause damage type to change. This damage type prediction will only use the following to determine potential damage types: - Intrinsic perks - Talent Node perks - Known, reusable plugs for sockets |
| `defaultDamageType` | int32 | If the item has a damage type that could be considered to be default, it will be populated here. For various upsetting reasons, it's surprisingly cumbersome to figure this out. I hope you're happy. |
| `defaultDamageTypeHash` | uint32 → DestinyDamageTypeDefinition? | Similar to defaultDamageType, but represented as the hash identifier for a DestinyDamageTypeDefinition. I will likely regret leaving in the enumeration versions of these properties, but for now they're very convenient. |
| `seasonHash` | uint32 → DestinySeasonDefinition? | If this item is related directly to a Season of Destiny, this is the hash identifier for that season. |
| `isWrapper` | boolean | If true, this is a dummy vendor-wrapped item template. Items purchased from Eververse will be "wrapped" by one of these items so that we can safely provide refund capabilities before the item is "unwrapped". |
| `traitIds` | array&lt;string&gt; | Traits are metadata tags applied to this item. For example: armor slot, weapon type, foundry, faction, etc. These IDs come from the game and don't map to any content, but should still be useful. |
| `traitHashes` | array&lt;uint32&gt; | These are the corresponding trait definition hashes for the entries in traitIds. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyItemTooltipNotification

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `displayString` | string | — |
| `displayStyle` | string | — |

#### Destiny.Misc.DestinyColor

**Type:** object

Represents a color whose RGBA values are all represented as values between 0 and 255.

| Property | Type | Description |
| --- | --- | --- |
| `red` | byte | — |
| `green` | byte | — |
| `blue` | byte | — |
| `alpha` | byte | — |

#### Destiny.Definitions.DestinyItemActionBlockDefinition

**Type:** object

If an item can have an action performed on it (like "Dismantle"), it will be defined here if you care.

| Property | Type | Description |
| --- | --- | --- |
| `verbName` | string | Localized text for the verb of the action being performed. |
| `verbDescription` | string | Localized text describing the action being performed. |
| `isPositive` | boolean | The content has this property, however it's not entirely clear how it is used. |
| `overlayScreenName` | string | If the action has an overlay screen associated with it, this is the name of that screen. Unfortunately, we cannot return the screen's data itself. |
| `overlayIcon` | string | The icon associated with the overlay screen for the action, if any. |
| `requiredCooldownSeconds` | int32 | The number of seconds to delay before allowing this action to be performed again. |
| `requiredItems` | array&lt;Destiny.Definitions.DestinyItemActionRequiredItemDefinition&gt; | If the action requires other items to exist or be destroyed, this is the list of those items and requirements. |
| `progressionRewards` | array&lt;Destiny.Definitions.DestinyProgressionRewardDefinition&gt; | If performing this action earns you Progression, this is the list of progressions and values granted for those progressions by performing this action. |
| `actionTypeLabel` | string | The internal identifier for the action. |
| `requiredLocation` | string | Theoretically, an item could have a localized string for a hint about the location in which the action should be performed. In practice, no items yet have this property. |
| `requiredCooldownHash` | uint32 | The identifier hash for the Cooldown associated with this action. We have not pulled this data yet for you to have more data to use for cooldowns. |
| `deleteOnAction` | boolean | If true, the item is deleted when the action completes. |
| `consumeEntireStack` | boolean | If true, the entire stack is deleted when the action completes. |
| `useOnAcquire` | boolean | If true, this action will be performed as soon as you earn this item. Some rewards work this way, providing you a single item to pick up from a reward-granting vendor in-game and then immediately consuming itself to provide you multiple items. |

#### Destiny.Definitions.DestinyItemActionRequiredItemDefinition

**Type:** object

The definition of an item and quantity required in a character's inventory in order to perform an action.

| Property | Type | Description |
| --- | --- | --- |
| `count` | int32 | The minimum quantity of the item you have to have. |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier of the item you need to have. Use it to look up the DestinyInventoryItemDefinition for more info. |
| `deleteOnAction` | boolean | If true, the item/quantity will be deleted from your inventory when the action is performed. Otherwise, you'll retain these required items after the action is complete. |

#### Destiny.Definitions.DestinyProgressionRewardDefinition

**Type:** object

Inventory Items can reward progression when actions are performed on them. A common example of this in Destiny 1 was Bounties, which would reward Experience on your Character and the like when you completed the bounty. Note that this maps to a DestinyProgressionMappingDefinition, and *not* a DestinyProgressionDefinition directly. This is apparently so that multiple progressions can be granted progression points/experience at the same time.

| Property | Type | Description |
| --- | --- | --- |
| `progressionMappingHash` | uint32 → DestinyProgressionMappingDefinition | The hash identifier of the DestinyProgressionMappingDefinition that contains the progressions for which experience should be applied. |
| `amount` | int32 | The amount of experience to give to each of the mapped progressions. |
| `applyThrottles` | boolean | If true, the game's internal mechanisms to throttle progression should be applied. |

#### Destiny.Definitions.DestinyProgressionMappingDefinition

**Type:** object

Aggregations of multiple progressions. These are used to apply rewards to multiple progressions at once. They can sometimes have human readable data as well, but only extremely sporadically.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | Infrequently defined in practice. Defer to the individual progressions' display properties. |
| `displayUnits` | string | The localized unit of measurement for progression across the progressions defined in this mapping. Unfortunately, this is very infrequently defined. Defer to the individual progressions' display units. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyItemCraftingBlockDefinition

**Type:** object

If an item can have an action performed on it (like "Dismantle"), it will be defined here if you care.

| Property | Type | Description |
| --- | --- | --- |
| `outputItemHash` | uint32 → DestinyInventoryItemDefinition | A reference to the item definition that is created when crafting with this 'recipe' item. |
| `requiredSocketTypeHashes` | array&lt;uint32&gt; → DestinySocketTypeDefinition | A list of socket type hashes that describes which sockets are required for crafting with this recipe. |
| `failedRequirementStrings` | array&lt;string&gt; | — |
| `baseMaterialRequirements` | uint32 → DestinyMaterialRequirementSetDefinition? | A reference to the base material requirements for crafting with this recipe. |
| `bonusPlugs` | array&lt;Destiny.Definitions.DestinyItemCraftingBlockBonusPlugDefinition&gt; | A list of 'bonus' socket plugs that may be available if certain requirements are met. |

#### Destiny.Definitions.DestinyItemCraftingBlockBonusPlugDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `socketTypeHash` | uint32 → DestinySocketTypeDefinition | — |
| `plugItemHash` | uint32 → DestinyInventoryItemDefinition | — |

#### Destiny.Definitions.Sockets.DestinySocketTypeDefinition

**Object** · *(Manifest definition, table `SocketTypes`)*

All Sockets have a "Type": a set of common properties that determine when the socket allows Plugs to be inserted, what Categories of Plugs can be inserted, and whether the socket is even visible at all given the current game/character/account state. See DestinyInventoryItemDefinition for more information about Socketed items and Plugs.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | There are fields for this display data, but they appear to be unpopulated as of now. I am not sure where in the UI these would show if they even were populated, but I will continue to return this data in case it becomes useful. |
| `insertAction` | Destiny.Definitions.Sockets.DestinyInsertPlugActionDefinition | Defines what happens when a plug is inserted into sockets of this type. |
| `plugWhitelist` | array&lt;Destiny.Definitions.Sockets.DestinyPlugWhitelistEntryDefinition&gt; | A list of Plug "Categories" that are allowed to be plugged into sockets of this type. These should be compared against a given plug item's DestinyInventoryItemDefinition.plug.plugCategoryHash, which indicates the plug item's category. If the plug's category matches any whitelisted plug, or if the whitelist is empty, it is allowed to be inserted. |
| `socketCategoryHash` | uint32 → DestinySocketCategoryDefinition | — |
| `visibility` | int32 | Sometimes a socket isn't visible. These are some of the conditions under which sockets of this type are not visible. Unfortunately, the truth of visibility is much, much more complex. Best to rely on the live data for whether the socket is visible and enabled. |
| `alwaysRandomizeSockets` | boolean | — |
| `isPreviewEnabled` | boolean | — |
| `hideDuplicateReusablePlugs` | boolean | — |
| `overridesUiAppearance` | boolean | This property indicates if the socket type determines whether Emblem icons and nameplates should be overridden by the inserted plug item's icon and nameplate. |
| `avoidDuplicatesOnInitialization` | boolean | — |
| `currencyScalars` | array&lt;Destiny.Definitions.Sockets.DestinySocketTypeScalarMaterialRequirementEntry&gt; | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Sockets.DestinyInsertPlugActionDefinition

**Type:** object

Data related to what happens while a plug is being inserted, mostly for UI purposes.

| Property | Type | Description |
| --- | --- | --- |
| `actionExecuteSeconds` | int32 | How long it takes for the Plugging of the item to be completed once it is initiated, if you care. |
| `actionType` | int32 | The type of action being performed when you act on this Socket Type. The most common value is "insert plug", but there are others as well (for instance, a "Masterwork" socket may allow for Re-initialization, and an Infusion socket allows for items to be consumed to upgrade the item) |

#### Destiny.SocketTypeActionTypeEnumeration

**Enum** (`int32`)

Indicates the type of actions that can be performed

| Value | # | Description |
| --- | --- | --- |
| `InsertPlug` | 0 | — |
| `InfuseItem` | 1 | — |
| `ReinitializeSocket` | 2 | — |

#### Destiny.Definitions.Sockets.DestinyPlugWhitelistEntryDefinition

**Type:** object

Defines a plug "Category" that is allowed to be plugged into a socket of this type. This should be compared against a given plug item's DestinyInventoryItemDefinition.plug.plugCategoryHash, which indicates the plug item's category.

| Property | Type | Description |
| --- | --- | --- |
| `categoryHash` | uint32 | The hash identifier of the Plug Category to compare against the plug item's plug.plugCategoryHash. Note that this does NOT relate to any Definition in itself, it is only used for comparison purposes. |
| `categoryIdentifier` | string | The string identifier for the category, which is here mostly for debug purposes. |
| `reinitializationPossiblePlugHashes` | array&lt;uint32&gt; | The list of all plug items (DestinyInventoryItemDefinition) that the socket may randomly be populated with when reinitialized. Which ones you should actually show are determined by the plug being inserted into the socket, and the socket’s type. When you inspect the plug that could go into a Masterwork Socket, look up the socket type of the socket being inspected and find the DestinySocketTypeDefinition. Then, look at the Plugs that can fit in that socket. Find the Whitelist in the DestinySocketTypeDefinition that matches the plug item’s categoryhash. That whitelist entry will potentially have a new “reinitializationPossiblePlugHashes” property.If it does, that means we know what it will roll if you try to insert this plug into this socket. |

#### Destiny.DestinySocketVisibilityEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Visible` | 0 | — |
| `Hidden` | 1 | — |
| `HiddenWhenEmpty` | 2 | — |
| `HiddenIfNoPlugsAvailable` | 3 | — |

#### Destiny.Definitions.Sockets.DestinySocketTypeScalarMaterialRequirementEntry

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `currencyItemHash` | uint32 → DestinyInventoryItemDefinition | — |
| `scalarValue` | int32 | — |

#### Destiny.Definitions.Sockets.DestinySocketCategoryDefinition

**Object** · *(Manifest definition, table `SocketCategories`)*

Sockets on an item are organized into Categories visually. You can find references to the socket category defined on an item's DestinyInventoryItemDefinition.sockets.socketCategories property. This has the display information for rendering the categories' header, and a hint for how the UI should handle showing this category. The shitty thing about this, however, is that the socket categories' UI style can be overridden by the item's UI style. For instance, the Socket Category used by Emote Sockets says it's "consumable," but that's a lie: they're all reusable, and overridden by the detail UI pages in ways that we can't easily account for in the API. As a result, I will try to compile these rules into the individual sockets on items, and provide the best hint possible there through the plugSources property. In the future, I may attempt to use this information in conjunction with the item to provide a more usable UI hint on the socket layer, but for now improving the consistency of plugSources is the best I have time to provide. (See https:// github.com/Bungie-net/api/issues/522 for more info)

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `uiCategoryStyle` | uint32 | A string hinting to the game's UI system about how the sockets in this category should be displayed. BNet doesn't use it: it's up to you to find valid values and make your own special UI if you want to honor this category style. |
| `categoryStyle` | int32 | Same as uiCategoryStyle, but in a more usable enumeration form. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.DestinySocketCategoryStyleEnumeration

**Enum** (`int32`)

Represents the possible and known UI styles used by the game for rendering Socket Categories.

| Value | # | Description |
| --- | --- | --- |
| `Unknown` | 0 | — |
| `Reusable` | 1 | — |
| `Consumable` | 2 | — |
| `Unlockable` | 3 | — |
| `Intrinsic` | 4 | — |
| `EnergyMeter` | 5 | — |
| `LargePerk` | 6 | — |
| `Abilities` | 7 | — |
| `Supers` | 8 | — |

#### Destiny.Definitions.DestinyMaterialRequirementSetDefinition

**Object** · *(Manifest definition, table `MaterialRequirementSets`)*

Represent a set of material requirements: Items that either need to be owned or need to be consumed in order to perform an action. A variety of other entities refer to these as gatekeepers and payments for actions that can be performed in game.

| Property | Type | Description |
| --- | --- | --- |
| `materials` | array&lt;Destiny.Definitions.DestinyMaterialRequirement&gt; | The list of all materials that are required. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyMaterialRequirement

**Type:** object

Many actions relating to items require you to expend materials: - Activating a talent node - Inserting a plug into a socket The items will refer to material requirements by a materialRequirementsHash in these cases, and this is the definition for those requirements in terms of the item required, how much of it is required and other interesting info. This is one of the rare/strange times where a single contract class is used both in definitions *and* in live data response contracts. I'm not sure yet whether I regret that.

| Property | Type | Description |
| --- | --- | --- |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier of the material required. Use it to look up the material's DestinyInventoryItemDefinition. |
| `deleteOnAction` | boolean | If True, the material will be removed from the character's inventory when the action is performed. |
| `count` | int32 | The amount of the material required. |
| `countIsConstant` | boolean | If true, the material requirement count value is constant. Since The Witch Queen expansion, some material requirement counts can be dynamic and will need to be returned with an API call. |
| `omitFromRequirements` | boolean | If True, this requirement is "silent": don't bother showing it in a material requirements display. I mean, I'm not your mom: I'm not going to tell you you *can't* show it. But we won't show it in our UI. |
| `hasVirtualStackSize` | boolean | If true, this material requirement references a virtual item stack size value. You can get that value from a corresponding DestinyMaterialRequirementSetState. |

#### Destiny.Definitions.DestinyItemInventoryBlockDefinition

**Type:** object

If the item can exist in an inventory - the overwhelming majority of them can and do - then this is the basic properties regarding the item's relationship with the inventory.

| Property | Type | Description |
| --- | --- | --- |
| `stackUniqueLabel` | string | If this string is populated, you can't have more than one stack with this label in a given inventory. Note that this is different from the equipping block's unique label, which is used for equipping uniqueness. |
| `maxStackSize` | int32 | The maximum quantity of this item that can exist in a stack. |
| `bucketTypeHash` | uint32 → DestinyInventoryBucketDefinition | The hash identifier for the DestinyInventoryBucketDefinition to which this item belongs. I should have named this "bucketHash", but too many things refer to it now. Sigh. |
| `recoveryBucketTypeHash` | uint32 → DestinyInventoryBucketDefinition | If the item is picked up by the lost loot queue, this is the hash identifier for the DestinyInventoryBucketDefinition into which it will be placed. Again, I should have named this recoveryBucketHash instead. |
| `tierTypeHash` | uint32 → DestinyItemTierTypeDefinition | The hash identifier for the Tier Type of the item, use to look up its DestinyItemTierTypeDefinition if you need to show localized data for the item's tier. |
| `isInstanceItem` | boolean | If TRUE, this item is instanced. Otherwise, it is a generic item that merely has a quantity in a stack (like Glimmer). |
| `tierTypeName` | string | The localized name of the tier type, which is a useful shortcut so you don't have to look up the definition every time. However, it's mostly a holdover from days before we had a DestinyItemTierTypeDefinition to refer to. |
| `tierType` | int32 | The enumeration matching the tier type of the item to known values, again for convenience sake. |
| `expirationTooltip` | string | The tooltip message to show, if any, when the item expires. |
| `expiredInActivityMessage` | string | If the item expires while playing in an activity, we show a different message. |
| `expiredInOrbitMessage` | string | If the item expires in orbit, we show a... more different message. ("Consummate V's, consummate!") |
| `suppressExpirationWhenObjectivesComplete` | boolean | — |
| `recipeItemHash` | uint32 → DestinyInventoryItemDefinition? | A reference to the associated crafting 'recipe' item definition, if this item can be crafted. |

#### Destiny.TierTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Unknown` | 0 | — |
| `Currency` | 1 | — |
| `Basic` | 2 | — |
| `Common` | 3 | — |
| `Rare` | 4 | — |
| `Superior` | 5 | — |
| `Exotic` | 6 | — |

#### Destiny.Definitions.DestinyInventoryBucketDefinition

**Object** · *(Manifest definition, table `InventoryBuckets`)*

An Inventory (be it Character or Profile level) is comprised of many Buckets. An example of a bucket is "Primary Weapons", where all of the primary weapons on a character are gathered together into a single visual element in the UI: a subset of the inventory that has a limited number of slots, and in this case also has an associated Equipment Slot for equipping an item in the bucket. Item definitions declare what their "default" bucket is (DestinyInventoryItemDefinition.inventory.bucketTypeHash), and Item instances will tell you which bucket they are currently residing in (DestinyItemComponent.bucketHash). You can use this information along with the DestinyInventoryBucketDefinition to show these items grouped by bucket. You cannot transfer an item to a bucket that is not its Default without going through a Vendor's "accepted items" (DestinyVendorDefinition.acceptedItems). This is how transfer functionality like the Vault is implemented, as a feature of a Vendor. See the vendor's acceptedItems property for more details.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `scope` | int32 | Where the bucket is found. 0 = Character, 1 = Account |
| `category` | int32 | An enum value for what items can be found in the bucket. See the BucketCategory enum for more details. |
| `bucketOrder` | int32 | Use this property to provide a quick-and-dirty recommended ordering for buckets in the UI. Most UIs will likely want to forsake this for something more custom and manual. |
| `itemCount` | int32 | The maximum # of item "slots" in a bucket. A slot is a given combination of item + quantity. For instance, a Weapon will always take up a single slot, and always have a quantity of 1. But a material could take up only a single slot with hundreds of quantity. |
| `location` | int32 | Sometimes, inventory buckets represent conceptual "locations" in the game that might not be expected. This value indicates the conceptual location of the bucket, regardless of where it is actually contained on the character/account. See ItemLocation for details. Note that location includes the Vault and the Postmaster (both of whom being just inventory buckets with additional actions that can be performed on them through a Vendor) |
| `hasTransferDestination` | boolean | If TRUE, there is at least one Vendor that can transfer items to/from this bucket. See the DestinyVendorDefinition's acceptedItems property for more information on how transferring works. |
| `enabled` | boolean | If True, this bucket is enabled. Disabled buckets may include buckets that were included for test purposes, or that were going to be used but then were abandoned but never removed from content *cough*. |
| `fifo` | boolean | if a FIFO bucket fills up, it will delete the oldest item from said bucket when a new item tries to be added to it. If this is FALSE, the bucket will not allow new items to be placed in it until room is made by the user manually deleting items from it. You can see an example of this with the Postmaster's bucket. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.BucketScopeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Character` | 0 | — |
| `Account` | 1 | — |

#### Destiny.BucketCategoryEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Invisible` | 0 | — |
| `Item` | 1 | — |
| `Currency` | 2 | — |
| `Equippable` | 3 | — |
| `Ignored` | 4 | — |

#### Destiny.ItemLocationEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Unknown` | 0 | — |
| `Inventory` | 1 | — |
| `Vault` | 2 | — |
| `Vendor` | 3 | — |
| `Postmaster` | 4 | — |

#### Destiny.Definitions.Items.DestinyItemTierTypeDefinition

**Object** · *(Manifest definition, table `ItemTierTypes`)*

Defines the tier type of an item. Mostly this provides human readable properties for types like Common, Rare, etc... It also provides some base data for infusion that could be useful.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `infusionProcess` | Destiny.Definitions.Items.DestinyItemTierTypeInfusionBlock | If this tier defines infusion properties, they will be contained here. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Items.DestinyItemTierTypeInfusionBlock

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `baseQualityTransferRatio` | float | The default portion of quality that will transfer from the infuser to the infusee item. (InfuserQuality - InfuseeQuality) \* baseQualityTransferRatio = base quality transferred. |
| `minimumQualityIncrement` | int32 | As long as InfuserQuality > InfuseeQuality, the amount of quality bestowed is guaranteed to be at least this value, even if the transferRatio would dictate that it should be less. The total amount of quality that ends up in the Infusee cannot exceed the Infuser's quality however (for instance, if you infuse a 300 item with a 301 item and the minimum quality increment is 10, the infused item will not end up with 310 quality) |

#### Destiny.Definitions.DestinyItemSetBlockDefinition

**Type:** object

Primarily for Quests, this is the definition of properties related to the item if it is a quest and its various quest steps.

| Property | Type | Description |
| --- | --- | --- |
| `itemList` | array&lt;Destiny.Definitions.DestinyItemSetBlockEntryDefinition&gt; | A collection of hashes of set items, for items such as Quest Metadata items that possess this data. |
| `requireOrderedSetItemAdd` | boolean | If true, items in the set can only be added in increasing order, and adding an item will remove any previous item. For Quests, this is by necessity true. Only one quest step is present at a time, and previous steps are removed as you advance in the quest. |
| `setIsFeatured` | boolean | If true, the UI should treat this quest as "featured" |
| `setType` | string | A string identifier we can use to attempt to identify the category of the Quest. |
| `questLineName` | string | The name of the quest line that this quest step is a part of. |
| `questLineDescription` | string | The description of the quest line that this quest step is a part of. |
| `questStepSummary` | string | An additional summary of this step in the quest line. |

#### Destiny.Definitions.DestinyItemSetBlockEntryDefinition

**Type:** object

Defines a particular entry in an ItemSet (AKA a particular Quest Step in a Quest)

| Property | Type | Description |
| --- | --- | --- |
| `trackingValue` | int32 | Used for tracking which step a user reached. These values will be populated in the user's internal state, which we expose externally as a more usable DestinyQuestStatus object. If this item has been obtained, this value will be set in trackingUnlockValueHash. |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | This is the hash identifier for a DestinyInventoryItemDefinition representing this quest step. |

#### Destiny.Definitions.DestinyItemStatBlockDefinition

**Type:** object

Information about the item's calculated stats, with as much data as we can find for the stats without having an actual instance of the item. Note that this means the entire concept of providing these stats is fundamentally insufficient: we cannot predict with 100% accuracy the conditions under which an item can spawn, so we use various heuristics to attempt to simulate the conditions as accurately as possible. Actual stats for items in- game can and will vary, but these should at least be useful base points for comparison and display. It is also worth noting that some stats, like Magazine size, have further calculations performed on them by scripts in-game and on the game servers that BNet does not have access to. We cannot know how those stats are further transformed, and thus some stats will be inaccurate even on instances of items in BNet vs. how they appear in-game. This is a known limitation of our item statistics, without any planned fix.

| Property | Type | Description |
| --- | --- | --- |
| `disablePrimaryStatDisplay` | boolean | If true, the game won't show the "primary" stat on this item when you inspect it. NOTE: This is being manually mapped, because I happen to want it in a block that isn't going to directly create this derivative block. |
| `statGroupHash` | uint32 → DestinyStatGroupDefinition? | If the item's stats are meant to be modified by a DestinyStatGroupDefinition, this will be the identifier for that definition. If you are using live data or precomputed stats data on the DestinyInventoryItemDefinition.stats.stats property, you don't have to worry about statGroupHash and how it alters stats: the already altered stats are provided to you. But if you want to see how the sausage gets made, or perform computations yourself, this is valuable information. |
| `stats` | Mapping&lt;uint32, Destiny.Definitions.DestinyInventoryItemStatDefinition&gt; → DestinyStatDefinition | If you are looking for precomputed values for the stats on a weapon, this is where they are stored. Technically these are the "Display" stat values. Please see DestinyStatsDefinition for what Display Stat Values means, it's a very long story... but essentially these are the closest values BNet can get to the item stats that you see in-game. These stats are keyed by the DestinyStatDefinition's hash identifier for the stat that's found on the item. |
| `hasDisplayableStats` | boolean | A quick and lazy way to determine whether any stat other than the "primary" stat is actually visible on the item. Items often have stats that we return in case people find them useful, but they're not part of the "Stat Group" and thus we wouldn't display them in our UI. If this is False, then we're not going to display any of these stats other than the primary one. |
| `primaryBaseStatHash` | uint32 → DestinyStatDefinition | This stat is determined to be the "primary" stat, and can be looked up in the stats or any other stat collection related to the item. Use this hash to look up the stat's value using DestinyInventoryItemDefinition.stats.stats, and the renderable data for the primary stat in the related DestinyStatDefinition. |

#### Destiny.Definitions.DestinyInventoryItemStatDefinition

**Type:** object

Defines a specific stat value on an item, and the minimum/maximum range that we could compute for the item based on our heuristics for how the item might be generated. Not guaranteed to match real-world instances of the item, but should hopefully at least be close. If it's not close, let us know on the Bungie API forums.

| Property | Type | Description |
| --- | --- | --- |
| `statHash` | uint32 → DestinyStatDefinition | The hash for the DestinyStatDefinition representing this stat. |
| `value` | int32 | This value represents the stat value assuming the minimum possible roll but accounting for any mandatory bonuses that should be applied to the stat on item creation. In Destiny 1, this was different from the "minimum" value because there were certain conditions where an item could be theoretically lower level/value than the initial roll. In Destiny 2, this is not possible unless Talent Grids begin to be used again for these purposes or some other system change occurs... thus in practice, value and minimum should be the same in Destiny 2. Good riddance. |
| `minimum` | int32 | The minimum possible value for this stat that we think the item can roll. |
| `maximum` | int32 | The maximum possible value for this stat that we think the item can roll. WARNING: In Destiny 1, this field was calculated using the potential stat rolls on the item's talent grid. In Destiny 2, items no longer have meaningful talent grids and instead have sockets: but the calculation of this field was never altered to adapt to this change. As such, this field should be considered deprecated until we can address this oversight. |
| `displayMaximum` | int32? | The maximum possible value for the stat as shown in the UI, if it is being shown somewhere that reveals maximum in the UI (such as a bar chart-style view). This is pulled directly from the item's DestinyStatGroupDefinition, and placed here for convenience. If not returned, there is no maximum to use (and thus the stat should not be shown in a way that assumes there is a limit to the stat) |

#### Destiny.Definitions.DestinyStatGroupDefinition

**Object** · *(Manifest definition, table `StatGroups`)*

When an inventory item (DestinyInventoryItemDefinition) has Stats (such as Attack Power), the item will refer to a Stat Group. This definition enumerates the properties used to transform the item's "Investment" stats into "Display" stats. See DestinyStatDefinition's documentation for information about the transformation of Stats, and the meaning of an Investment vs. a Display stat. If you don't want to do these calculations on your own, fear not: pulling live data from the BNet endpoints will return display stat values pre-computed and ready for you to use. I highly recommend this approach, saves a lot of time and also accounts for certain stat modifiers that can't easily be accounted for without live data (such as stat modifiers on Talent Grids and Socket Plugs)

| Property | Type | Description |
| --- | --- | --- |
| `maximumValue` | int32 | The maximum possible value that any stat in this group can be transformed into. This is used by stats that *don't* have scaledStats entries below, but that still need to be displayed as a progress bar, in which case this is used as the upper bound for said progress bar. (the lower bound is always 0) |
| `uiPosition` | int32 | This apparently indicates the position of the stats in the UI? I've returned it in case anyone can use it, but it's not of any use to us on BNet. Something's being lost in translation with this value. |
| `scaledStats` | array&lt;Destiny.Definitions.DestinyStatDisplayDefinition&gt; | Any stat that requires scaling to be transformed from an "Investment" stat to a "Display" stat will have an entry in this list. For more information on what those types of stats mean and the transformation process, see DestinyStatDefinition. In retrospect, I wouldn't mind if this was a dictionary keyed by the stat hash instead. But I'm going to leave it be because [[After Apple Picking]]. |
| `overrides` | Mapping&lt;uint32, Destiny.Definitions.DestinyStatOverrideDefinition&gt; | The game has the ability to override, based on the stat group, what the localized text is that is displayed for Stats being shown on the item. Mercifully, no Stat Groups use this feature currently. If they start using them, we'll all need to start using them (and those of you who are more prudent than I am can go ahead and start pre- checking for this.) |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyStatDisplayDefinition

**Type:** object

Describes the way that an Item Stat (see DestinyStatDefinition) is transformed using the DestinyStatGroupDefinition related to that item. See both of the aforementioned definitions for more information about the stages of stat transformation. This represents the transformation of a stat into a "Display" stat (the closest value that BNet can get to the in-game display value of the stat)

| Property | Type | Description |
| --- | --- | --- |
| `statHash` | uint32 → DestinyStatDefinition | The hash identifier for the stat being transformed into a Display stat. Use it to look up the DestinyStatDefinition, or key into a DestinyInventoryItemDefinition's stats property. |
| `maximumValue` | int32 | Regardless of the output of interpolation, this is the maximum possible value that the stat can be. It should also be used as the upper bound for displaying the stat as a progress bar (the minimum always being 0) |
| `displayAsNumeric` | boolean | If this is true, the stat should be displayed as a number. Otherwise, display it as a progress bar. Or, you know, do whatever you want. There's no displayAsNumeric police. |
| `displayInterpolation` | array&lt;Interpolation.InterpolationPoint&gt; | The interpolation table representing how the Investment Stat is transformed into a Display Stat. See DestinyStatDefinition for a description of the stages of stat transformation. |

#### Destiny.Definitions.DestinyStatOverrideDefinition

**Type:** object

Stat Groups (DestinyStatGroupDefinition) has the ability to override the localized text associated with stats that are to be shown on the items with which they are associated. This defines a specific overridden stat. You could theoretically check these before rendering your stat UI, and for each stat that has an override show these displayProperties instead of those on the DestinyStatDefinition. Or you could be like us, and skip that for now because the game has yet to actually use this feature. But know that it's here, waiting for a resilliant young designer to take up the mantle and make us all look foolish by showing the wrong name for stats. Note that, if this gets used, the override will apply only to items using the overriding Stat Group. Other items will still show the default stat's name/description.

| Property | Type | Description |
| --- | --- | --- |
| `statHash` | uint32 → DestinyStatDefinition | The hash identifier of the stat whose display properties are being overridden. |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | The display properties to show instead of the base DestinyStatDefinition display properties. |

#### Destiny.Definitions.DestinyEquippingBlockDefinition

**Type:** object

Items that can be equipped define this block. It contains information we need to understand how and when the item can be equipped.

| Property | Type | Description |
| --- | --- | --- |
| `gearsetItemHash` | uint32 → DestinyInventoryItemDefinition? | If the item is part of a gearset, this is a reference to that gearset item. |
| `uniqueLabel` | string | If defined, this is the label used to check if the item has other items of matching types already equipped. For instance, when you aren't allowed to equip more than one Exotic Weapon, that's because all exotic weapons have identical uniqueLabels and the game checks the to-be-equipped item's uniqueLabel vs. all other already equipped items (other than the item in the slot that's about to be occupied). |
| `uniqueLabelHash` | uint32 | The hash of that unique label. Does not point to a specific definition. |
| `equipmentSlotTypeHash` | uint32 → DestinyEquipmentSlotDefinition | An equipped item *must* be equipped in an Equipment Slot. This is the hash identifier of the DestinyEquipmentSlotDefinition into which it must be equipped. |
| `attributes` | int32 | These are custom attributes on the equippability of the item. For now, this can only be "equip on acquire", which would mean that the item will be automatically equipped as soon as you pick it up. |
| `ammoType` | int32 | Ammo type used by a weapon is no longer determined by the bucket in which it is contained. If the item has an ammo type - i.e. if it is a weapon - this will be the type of ammunition expected. |
| `displayStrings` | array&lt;string&gt; | These are strings that represent the possible Game/Account/Character state failure conditions that can occur when trying to equip the item. They match up one-to-one with requiredUnlockExpressions. |
| `equipableItemSetHash` | uint32 → DestinyEquipableItemSetDefinition? | If this item is part of an item set with bonus perks, this will the hash of that set. |

#### Destiny.EquippingItemBlockAttributesEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `EquipOnAcquire` | 1 | — |

#### Destiny.DestinyAmmunitionTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Primary` | 1 | — |
| `Special` | 2 | — |
| `Heavy` | 3 | — |
| `Unknown` | 4 | — |

#### Destiny.Definitions.DestinyEquipmentSlotDefinition

**Object** · *(Manifest definition, table `EquipmentSlots`)*

Characters can not only have Inventory buckets (containers of items that are generally matched by their type or functionality), they can also have Equipment Slots. The Equipment Slot is an indicator that the related bucket can have instanced items equipped on the character. For instance, the Primary Weapon bucket has an Equipment Slot that determines whether you can equip primary weapons, and holds the association between its slot and the inventory bucket from which it can have items equipped. An Equipment Slot must have a related Inventory Bucket, but not all inventory buckets must have Equipment Slots.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `equipmentCategoryHash` | uint32 | These technically point to "Equipment Category Definitions". But don't get excited. There's nothing of significant value in those definitions, so I didn't bother to expose them. You can use the hash here to group equipment slots by common functionality, which serves the same purpose as if we had the Equipment Category definitions exposed. |
| `bucketTypeHash` | uint32 → DestinyInventoryBucketDefinition | The inventory bucket that owns this equipment slot. |
| `applyCustomArtDyes` | boolean | If True, equipped items should have their custom art dyes applied when rendering the item. Otherwise, custom art dyes on an item should be ignored if the item is equipped in this slot. |
| `artDyeChannels` | array&lt;Destiny.Definitions.DestinyArtDyeReference&gt; | The Art Dye Channels that apply to this equipment slot. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyArtDyeReference

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `artDyeChannelHash` | uint32 | — |

#### Destiny.Definitions.Items.DestinyEquipableItemSetDefinition

**Object** · *(Manifest definition, table `EquipableItemSets`)*

Perks that are active only when you have a certain number of set items equipped.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | Display Properties, including name and icon, for this item set |
| `setItems` | array&lt;uint32&gt; → DestinyInventoryItemDefinition | The items that confer these perks. |
| `setPerks` | array&lt;Destiny.Definitions.Items.DestinyItemSetPerkDefinition&gt; | The perks conferred by this set of armor pieces. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Items.DestinyItemSetPerkDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `requiredSetCount` | int32 | The number of set pieces required to activate the perk. |
| `sandboxPerkHash` | uint32 → DestinySandboxPerkDefinition | The perk this set confers. |

#### Destiny.Definitions.DestinySandboxPerkDefinition

**Object** · *(Manifest definition, table `SandboxPerks`)*

Perks are modifiers to a character or item that can be applied situationally. - Perks determine a weapon's damage type. - Perks put the Mods in Modifiers (they are literally the entity that bestows the Sandbox benefit for whatever fluff text about the modifier in the Socket, Plug or Talent Node) - Perks are applied for unique alterations of state in Objectives Anyways, I'm sure you can see why perks are so interesting. What Perks often don't have is human readable information, so we attempt to reverse engineer that by pulling that data from places that uniquely refer to these perks: namely, Talent Nodes and Plugs. That only gives us a subset of perks that are human readable, but those perks are the ones people generally care about anyways. The others are left as a mystery, their true purpose mostly unknown and undocumented.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | These display properties are by no means guaranteed to be populated. Usually when it is, it's only because we back-filled them with the displayProperties of some Talent Node or Plug item that happened to be uniquely providing that perk. |
| `perkIdentifier` | string | The string identifier for the perk. |
| `isDisplayable` | boolean | If true, you can actually show the perk in the UI. Otherwise, it doesn't have useful player-facing information. |
| `damageType` | int32 | If this perk grants a damage type to a weapon, the damage type will be defined here. Unless you have a compelling reason to use this enum value, use the damageTypeHash instead to look up the actual DestinyDamageTypeDefinition. |
| `damageTypeHash` | uint32 → DestinyDamageTypeDefinition? | The hash identifier for looking up the DestinyDamageTypeDefinition, if this perk has a damage type. This is preferred over using the damageType enumeration value, which has been left purely because it is occasionally convenient. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.DamageTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Kinetic` | 1 | — |
| `Arc` | 2 | — |
| `Thermal` | 3 | — |
| `Void` | 4 | — |
| `Raid` | 5 | — |
| `Stasis` | 6 | — |
| `Strand` | 7 | — |

#### Destiny.Definitions.DestinyDamageTypeDefinition

**Object** · *(Manifest definition, table `DamageTypes`)*

All damage types that are possible in the game are defined here, along with localized info and icons as needed.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | The description of the damage type, icon etc... |
| `transparentIconPath` | string | A variant of the icon that is transparent and colorless. |
| `showIcon` | boolean | If TRUE, the game shows this damage type's icon. Otherwise, it doesn't. Whether you show it or not is up to you. |
| `enumValue` | int32 | We have an enumeration for damage types for quick reference. This is the current definition's damage type enum value. |
| `color` | Destiny.Misc.DestinyColor | A color associated with the damage type. The displayProperties icon is tinted with a color close to this. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyItemTranslationBlockDefinition

**Type:** object

This Block defines the rendering data associated with the item, if any.

| Property | Type | Description |
| --- | --- | --- |
| `weaponPatternIdentifier` | string | — |
| `weaponPatternHash` | uint32 → DestinySandboxPatternDefinition | — |
| `defaultDyes` | array&lt;Destiny.DyeReference&gt; | — |
| `lockedDyes` | array&lt;Destiny.DyeReference&gt; | — |
| `customDyes` | array&lt;Destiny.DyeReference&gt; | — |
| `arrangements` | array&lt;Destiny.Definitions.DestinyGearArtArrangementReference&gt; | — |
| `hasGeometry` | boolean | — |

#### Destiny.DyeReference

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `channelHash` | uint32 | — |
| `dyeHash` | uint32 | — |

#### Destiny.Definitions.DestinyGearArtArrangementReference

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `classHash` | uint32 → DestinyClassDefinition | — |
| `artArrangementHash` | uint32 | — |

#### Destiny.Definitions.DestinyClassDefinition

**Object** · *(Manifest definition, table `Classes`)*

Defines a Character Class in Destiny 2. These are types of characters you can play, like Titan, Warlock, and Hunter.

| Property | Type | Description |
| --- | --- | --- |
| `classType` | int32 | In Destiny 1, we added a convenience Enumeration for referring to classes. We've kept it, though mostly for posterity. This is the enum value for this definition's class. |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `genderedClassNames` | Mapping&lt;int32, string&gt; | A localized string referring to the singular form of the Class's name when referred to in gendered form. Keyed by the DestinyGender. |
| `genderedClassNamesByGenderHash` | Mapping&lt;uint32, string&gt; → DestinyGenderDefinition | — |
| `mentorVendorHash` | uint32 → DestinyVendorDefinition? | Mentors don't really mean anything anymore. Don't expect this to be populated. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.DestinyClassEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Titan` | 0 | — |
| `Hunter` | 1 | — |
| `Warlock` | 2 | — |
| `Unknown` | 3 | — |

#### Destiny.DestinyGenderEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Male` | 0 | — |
| `Female` | 1 | — |
| `Unknown` | 2 | — |

#### Destiny.Definitions.DestinyGenderDefinition

**Object** · *(Manifest definition, table `Genders`)*

Gender is a social construct, and as such we have definitions for Genders. Right now there happens to only be two, but we'll see what the future holds.

| Property | Type | Description |
| --- | --- | --- |
| `genderType` | int32 | This is a quick reference enumeration for all of the currently defined Genders. We use the enumeration for quicker lookups in related data, like DestinyClassDefinition.genderedClassNames. |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyVendorDefinition

**Object** · *(Manifest definition, table `Vendors`)*

These are the definitions for Vendors. In Destiny, a Vendor can be a lot of things - some things that you wouldn't expect, and some things that you don't even see directly in the game. Vendors are the Dolly Levi of the Destiny universe. - Traditional Vendors as you see in game: people who you come up to and who give you quests, rewards, or who you can buy things from. - Kiosks/Collections, which are really just Vendors that don't charge currency (or charge some pittance of a currency) and whose gating for purchases revolves more around your character's state. - Previews for rewards or the contents of sacks. These are implemented as Vendors, where you can't actually purchase from them but the items that they have for sale and the categories of sale items reflect the rewards or contents of the sack. This is so that the game could reuse the existing Vendor display UI for rewards and save a bunch of wheel reinvention. - Item Transfer capabilities, like the Vault and Postmaster. Vendors can have "acceptedItem" buckets that determine the source and destination buckets for transfers. When you interact with such a vendor, these buckets are what gets shown in the UI instead of any items that the Vendor would have for sale. Yep, the Vault is a vendor. It is pretty much guaranteed that they'll be used for even more features in the future. They have come to be seen more as generic categorized containers for items than "vendors" in a traditional sense, for better or worse. Where possible and time allows, we'll attempt to split those out into their own more digestible derived "Definitions": but often time does not allow that, as you can see from the above ways that vendors are used which we never split off from Vendor Definitions externally. Since Vendors are so many things to so many parts of the game, the definition is understandably complex. You will want to combine this data with live Vendor information from the API when it is available.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.DestinyVendorDisplayPropertiesDefinition | — |
| `vendorProgressionType` | int32 | The type of reward progression that this vendor has. Default - The original rank progression from token redemption. Ritual - Progression from ranks in ritual content. For example: Crucible (Shaxx), Gambit (Drifter), and Battlegrounds (War Table). |
| `buyString` | string | If the vendor has a custom localized string describing the "buy" action, that is returned here. |
| `sellString` | string | Ditto for selling. Not that you can sell items to a vendor anymore. Will it come back? Who knows. The string's still there. |
| `displayItemHash` | uint32 → DestinyInventoryItemDefinition | If the vendor has an item that should be displayed as the "featured" item, this is the hash identifier for that DestinyVendorItemDefinition. Apparently this is usually a related currency, like a reputation token. But it need not be restricted to that. |
| `inhibitBuying` | boolean | If this is true, you aren't allowed to buy whatever the vendor is selling. |
| `inhibitSelling` | boolean | If this is true, you're not allowed to sell whatever the vendor is buying. |
| `factionHash` | uint32 → DestinyFactionDefinition | If the Vendor has a faction, this hash will be valid and point to a DestinyFactionDefinition. The game UI and BNet often mine the faction definition for additional elements and details to place on the screen, such as the faction's Progression status (aka "Reputation"). |
| `resetIntervalMinutes` | int32 | A number used for calculating the frequency of a vendor's inventory resetting/refreshing. Don't worry about calculating this - we do it on the server side and send you the next refresh date with the live data. |
| `resetOffsetMinutes` | int32 | Again, used for reset/refreshing of inventory. Don't worry too much about it. Unless you want to. |
| `failureStrings` | array&lt;string&gt; | If an item can't be purchased from the vendor, there may be many "custom"/game state specific reasons why not. This is a list of localized strings with messages for those custom failures. The live BNet data will return a failureIndexes property for items that can't be purchased: using those values to index into this array, you can show the user the appropriate failure message for the item that can't be bought. |
| `unlockRanges` | array&lt;Dates.DateRange&gt; | If we were able to predict the dates when this Vendor will be visible/available, this will be the list of those date ranges. Sadly, we're not able to predict this very frequently, so this will often be useless data. |
| `vendorIdentifier` | string | The internal identifier for the Vendor. A holdover from the old days of Vendors, but we don't have time to refactor it away. |
| `vendorPortrait` | string | A portrait of the Vendor's smiling mug. Or frothing tentacles. |
| `vendorBanner` | string | If the vendor has a custom banner image, that can be found here. |
| `enabled` | boolean | If a vendor is not enabled, we won't even save the vendor's definition, and we won't return any items or info about them. It's as if they don't exist. |
| `visible` | boolean | If a vendor is not visible, we still have and will give vendor definition info, but we won't use them for things like Advisors or UI. |
| `vendorSubcategoryIdentifier` | string | The identifier of the VendorCategoryDefinition for this vendor's subcategory. |
| `consolidateCategories` | boolean | If TRUE, consolidate categories that only differ by trivial properties (such as having minor differences in name) |
| `actions` | array&lt;Destiny.Definitions.DestinyVendorActionDefinition&gt; | Describes "actions" that can be performed on a vendor. Currently, none of these exist. But theoretically a Vendor could let you interact with it by performing actions. We'll see what these end up looking like if they ever get used. |
| `categories` | array&lt;Destiny.Definitions.DestinyVendorCategoryEntryDefinition&gt; | These are the headers for sections of items that the vendor is selling. When you see items organized by category in the header, it is these categories that it is showing. Well, technically not *exactly* these. On BNet, it doesn't make sense to have categories be "paged" as we do in Destiny, so we run some heuristics to attempt to aggregate pages of categories together. These are the categories post-concatenation, if the vendor had concatenation applied. If you want the pre-aggregated category data, use originalCategories. |
| `originalCategories` | array&lt;Destiny.Definitions.DestinyVendorCategoryEntryDefinition&gt; | See the categories property for a description of categories and why originalCategories exists. |
| `displayCategories` | array&lt;Destiny.Definitions.DestinyDisplayCategoryDefinition&gt; | Display Categories are different from "categories" in that these are specifically for visual grouping and display of categories in Vendor UI. The "categories" structure is for validation of the contained items, and can be categorized entirely separately from "Display Categories", there need be and often will be no meaningful relationship between the two. |
| `interactions` | array&lt;Destiny.Definitions.DestinyVendorInteractionDefinition&gt; | In addition to selling items, vendors can have "interactions": UI where you "talk" with the vendor and they offer you a reward, some item, or merely acknowledge via dialog that you did something cool. |
| `inventoryFlyouts` | array&lt;Destiny.Definitions.DestinyVendorInventoryFlyoutDefinition&gt; | If the vendor shows you items from your own inventory - such as the Vault vendor does - this data describes the UI around showing those inventory buckets and which ones get shown. |
| `itemList` | array&lt;Destiny.Definitions.DestinyVendorItemDefinition&gt; | If the vendor sells items (or merely has a list of items to show like the "Sack" vendors do), this is the list of those items that the vendor can sell. From this list, only a subset will be available from the vendor at any given time, selected randomly and reset on the vendor's refresh interval. Note that a vendor can sell the same item multiple ways: for instance, nothing stops a vendor from selling you some specific weapon but using two different currencies, or the same weapon at multiple "item levels". |
| `services` | array&lt;Destiny.Definitions.DestinyVendorServiceDefinition&gt; | BNet doesn't use this data yet, but it appears to be an optional list of flavor text about services that the Vendor can provide. |
| `acceptedItems` | array&lt;Destiny.Definitions.DestinyVendorAcceptedItemDefinition&gt; | If the Vendor is actually a vehicle for the transferring of items (like the Vault and Postmaster vendors), this defines the list of source->destination buckets for transferring. |
| `returnWithVendorRequest` | boolean | As many of you know, Vendor data has historically been pretty brutal on the BNet servers. In an effort to reduce this workload, only Vendors with this flag set will be returned on Vendor requests. This allows us to filter out Vendors that don't dynamic data that's particularly useful: things like "Preview/Sack" vendors, for example, that you can usually suss out the details for using just the definitions themselves. |
| `locations` | array&lt;Destiny.Definitions.Vendors.DestinyVendorLocationDefinition&gt; | A vendor can be at different places in the world depending on the game/character/account state. This is the list of possible locations for the vendor, along with conditions we use to determine which one is currently active. |
| `groups` | array&lt;Destiny.Definitions.DestinyVendorGroupReference&gt; | A vendor can be a part of 0 or 1 "groups" at a time: a group being a collection of Vendors related by either location or function/purpose. It's used for our our Companion Vendor UI. Only one of these can be active for a Vendor at a time. |
| `ignoreSaleItemHashes` | array&lt;uint32&gt; | Some items don't make sense to return in the API, for example because they represent an action to be performed rather than an item being sold. I'd rather we not do this, but at least in the short term this is a workable workaround. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyVendorDisplayPropertiesDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `largeIcon` | string | I regret calling this a "large icon". It's more like a medium-sized image with a picture of the vendor's mug on it, trying their best to look cool. Not what one would call an icon. |
| `subtitle` | string | — |
| `originalIcon` | string | If we replaced the icon with something more glitzy, this is the original icon that the vendor had according to the game's content. It may be more lame and/or have less razzle-dazzle. But who am I to tell you which icon to use. |
| `requirementsDisplay` | array&lt;Destiny.Definitions.DestinyVendorRequirementDisplayEntryDefinition&gt; | Vendors, in addition to expected display property data, may also show some "common requirements" as statically defined definition data. This might be when a vendor accepts a single type of currency, or when the currency is unique to the vendor and the designers wanted to show that currency when you interact with the vendor. |
| `smallTransparentIcon` | string | This is the icon used in parts of the game UI such as the vendor's waypoint. |
| `mapIcon` | string | This is the icon used in the map overview, when the vendor is located on the map. |
| `largeTransparentIcon` | string | This is apparently the "Watermark". I am not certain offhand where this is actually used in the Game UI, but some people may find it useful. |
| `description` | string | — |
| `name` | string | — |
| `icon` | string | Note that "icon" is sometimes misleading, and should be interpreted in the context of the entity. For instance, in Destiny 1 the DestinyRecordBookDefinition's icon was a big picture of a book. But usually, it will be a small square image that you can use as... well, an icon. They are currently represented as 96px x 96px images. |
| `iconHash` | uint32 → DestinyIconDefinition | — |
| `iconSequences` | array&lt;Destiny.Definitions.Common.DestinyIconSequenceDefinition&gt; | — |
| `highResIcon` | string | If this item has a high-res icon (at least for now, many things won't), then the path to that icon will be here. |
| `hasIcon` | boolean | — |

#### Destiny.Definitions.DestinyVendorRequirementDisplayEntryDefinition

**Type:** object

The localized properties of the requirementsDisplay, allowing information about the requirement or item being featured to be seen.

| Property | Type | Description |
| --- | --- | --- |
| `icon` | string | — |
| `name` | string | — |
| `source` | string | — |
| `type` | string | — |

#### Destiny.DestinyVendorProgressionTypeEnumeration

**Enum** (`int32`)

Describes the type of progression that a vendor has.

| Value | # | Description |
| --- | --- | --- |
| `Default` | 0 | The original rank progression from token redemption. |
| `Ritual` | 1 | Progression from ranks in ritual content. For example: Crucible (Shaxx), Gambit (Drifter), and Season 13 Battlegrounds (War Table). |
| `NoSeasonalRefresh` | 2 | A vendor progression with no seasonal refresh. For example: Xur in the Eternity destination for the 30th Anniversary. |

#### Destiny.Definitions.DestinyVendorActionDefinition

**Type:** object

If a vendor can ever end up performing actions, these are the properties that will be related to those actions. I'm not going to bother documenting this yet, as it is unused and unclear if it will ever be used... but in case it is ever populated and someone finds it useful, it is defined here.

| Property | Type | Description |
| --- | --- | --- |
| `description` | string | — |
| `executeSeconds` | int32 | — |
| `icon` | string | — |
| `name` | string | — |
| `verb` | string | — |
| `isPositive` | boolean | — |
| `actionId` | string | — |
| `actionHash` | uint32 | — |
| `autoPerformAction` | boolean | — |

#### Destiny.Definitions.DestinyVendorCategoryEntryDefinition

**Type:** object

This is the definition for a single Vendor Category, into which Sale Items are grouped.

| Property | Type | Description |
| --- | --- | --- |
| `categoryIndex` | int32 | The index of the category in the original category definitions for the vendor. |
| `sortValue` | int32 | Used in sorting items in vendors... but there's a lot more to it. Just go with the order provided in the itemIndexes property on the DestinyVendorCategoryComponent instead, it should be more reliable than trying to recalculate it yourself. |
| `categoryHash` | uint32 | The hashed identifier for the category. |
| `quantityAvailable` | int32 | The amount of items that will be available when this category is shown. |
| `showUnavailableItems` | boolean | If items aren't up for sale in this category, should we still show them (greyed out)? |
| `hideIfNoCurrency` | boolean | If you don't have the currency required to buy items from this category, should the items be hidden? |
| `hideFromRegularPurchase` | boolean | True if this category doesn't allow purchases. |
| `buyStringOverride` | string | The localized string for making purchases from this category, if it is different from the vendor's string for purchasing. |
| `disabledDescription` | string | If the category is disabled, this is the localized description to show. |
| `displayTitle` | string | The localized title of the category. |
| `overlay` | Destiny.Definitions.DestinyVendorCategoryOverlayDefinition | If this category has an overlay prompt that should appear, this contains the details of that prompt. |
| `vendorItemIndexes` | array&lt;int32&gt; | A shortcut for the vendor item indexes sold under this category. Saves us from some expensive reorganization at runtime. |
| `isPreview` | boolean | Sometimes a category isn't actually used to sell items, but rather to preview them. This implies different UI (and manual placement of the category in the UI) in the game, and special treatment. |
| `isDisplayOnly` | boolean | If true, this category only displays items: you can't purchase anything in them. |
| `resetIntervalMinutesOverride` | int32 | — |
| `resetOffsetMinutesOverride` | int32 | — |

#### Destiny.Definitions.DestinyVendorCategoryOverlayDefinition

**Type:** object

The details of an overlay prompt to show to a user. They are all fairly self-explanatory localized strings that can be shown.

| Property | Type | Description |
| --- | --- | --- |
| `choiceDescription` | string | — |
| `description` | string | — |
| `icon` | string | — |
| `title` | string | — |
| `currencyItemHash` | uint32 → DestinyInventoryItemDefinition? | If this overlay has a currency item that it features, this is said featured item. |

#### Destiny.Definitions.DestinyDisplayCategoryDefinition

**Type:** object

Display Categories are different from "categories" in that these are specifically for visual grouping and display of categories in Vendor UI. The "categories" structure is for validation of the contained items, and can be categorized entirely separately from "Display Categories", there need be and often will be no meaningful relationship between the two.

| Property | Type | Description |
| --- | --- | --- |
| `index` | int32 | — |
| `identifier` | string | A string identifier for the display category. |
| `displayCategoryHash` | uint32 | — |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `displayInBanner` | boolean | If true, this category should be displayed in the "Banner" section of the vendor's UI. |
| `progressionHash` | uint32 → DestinyProgressionDefinition? | If it exists, this is the hash identifier of a DestinyProgressionDefinition that represents the progression to show on this display category. Specific categories can now have thier own distinct progression, apparently. So that's cool. |
| `sortOrder` | int32 | If this category sorts items in a nonstandard way, this will be the way we sort. |
| `displayStyleHash` | uint32? | An indicator of how the category will be displayed in the UI. It's up to you to do something cool or interesting in response to this, or just to treat it as a normal category. |
| `displayStyleIdentifier` | string | An indicator of how the category will be displayed in the UI. It's up to you to do something cool or interesting in response to this, or just to treat it as a normal category. |

#### Destiny.VendorDisplayCategorySortOrderEnumeration

**Enum** (`int32`)

Display categories can have custom sort orders. These are the possible options.

| Value | # | Description |
| --- | --- | --- |
| `Default` | 0 | — |
| `SortByTier` | 1 | — |

#### Destiny.Definitions.DestinyVendorInteractionDefinition

**Type:** object

A Vendor Interaction is a dialog shown by the vendor other than sale items or transfer screens. The vendor is showing you something, and asking you to reply to it by choosing an option or reward.

| Property | Type | Description |
| --- | --- | --- |
| `interactionIndex` | int32 | The position of this interaction in its parent array. Note that this is NOT content agnostic, and should not be used as such. |
| `replies` | array&lt;Destiny.Definitions.DestinyVendorInteractionReplyDefinition&gt; | The potential replies that the user can make to the interaction. |
| `vendorCategoryIndex` | int32 | If >= 0, this is the category of sale items to show along with this interaction dialog. |
| `questlineItemHash` | uint32 → DestinyInventoryItemDefinition | If this interaction dialog is about a quest, this is the questline related to the interaction. You can use this to show the quest overview, or even the character's status with the quest if you use it to find the character's current Quest Step by checking their inventory against this questlineItemHash's DestinyInventoryItemDefinition.setData. |
| `sackInteractionList` | array&lt;Destiny.Definitions.DestinyVendorInteractionSackEntryDefinition&gt; | If this interaction is meant to show you sacks, this is the list of types of sacks to be shown. If empty, the interaction is not meant to show sacks. |
| `uiInteractionType` | uint32 | A UI hint for the behavior of the interaction screen. This is useful to determine what type of interaction is occurring, such as a prompt to receive a rank up reward or a prompt to choose a reward for completing a quest. The hash isn't as useful as the Enum in retrospect, well what can you do. Try using interactionType instead. |
| `interactionType` | int32 | The enumerated version of the possible UI hints for vendor interactions, which is a little easier to grok than the hash found in uiInteractionType. |
| `rewardBlockLabel` | string | If this interaction is displaying rewards, this is the text to use for the header of the reward- displaying section of the interaction. |
| `rewardVendorCategoryIndex` | int32 | If the vendor's reward list is sourced from one of his categories, this is the index into the category array of items to show. |
| `flavorLineOne` | string | If the vendor interaction has flavor text, this is some of it. |
| `flavorLineTwo` | string | If the vendor interaction has flavor text, this is the rest of it. |
| `headerDisplayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | The header for the interaction dialog. |
| `instructions` | string | The localized text telling the player what to do when they see this dialog. |

#### Destiny.Definitions.DestinyVendorInteractionReplyDefinition

**Type:** object

When the interaction is replied to, Reward sites will fire and items potentially selected based on whether the given unlock expression is TRUE. You can potentially choose one from multiple replies when replying to an interaction: this is how you get either/or rewards from vendors.

| Property | Type | Description |
| --- | --- | --- |
| `itemRewardsSelection` | int32 | The rewards granted upon responding to the vendor. |
| `reply` | string | The localized text for the reply. |
| `replyType` | int32 | An enum indicating the type of reply being made. |

#### Destiny.DestinyVendorInteractionRewardSelectionEnumeration

**Enum** (`int32`)

When a Vendor Interaction provides rewards, they'll either let you choose one or let you have all of them. This determines which it will be.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `One` | 1 | — |
| `All` | 2 | — |

#### Destiny.DestinyVendorReplyTypeEnumeration

**Enum** (`int32`)

This determines the type of reply that a Vendor will have during an Interaction.

| Value | # | Description |
| --- | --- | --- |
| `Accept` | 0 | — |
| `Decline` | 1 | — |
| `Complete` | 2 | — |

#### Destiny.Definitions.DestinyVendorInteractionSackEntryDefinition

**Type:** object

Compare this sackType to the sack identifier in the DestinyInventoryItemDefinition.vendorSackType property of items. If they match, show this sack with this interaction.

| Property | Type | Description |
| --- | --- | --- |
| `sackType` | uint32 | — |

#### Destiny.VendorInteractionTypeEnumeration

**Enum** (`int32`)

An enumeration of the known UI interactions for Vendors.

| Value | # | Description |
| --- | --- | --- |
| `Unknown` | 0 | — |
| `Undefined` | 1 | An empty interaction. If this ends up in content, it is probably a game bug. |
| `QuestComplete` | 2 | An interaction shown when you complete a quest and receive a reward. |
| `QuestContinue` | 3 | An interaction shown when you talk to a Vendor as an intermediary step of a quest. |
| `ReputationPreview` | 4 | An interaction shown when you are previewing the vendor's reputation rewards. |
| `RankUpReward` | 5 | An interaction shown when you rank up with the vendor. |
| `TokenTurnIn` | 6 | An interaction shown when you have tokens to turn in for the vendor. |
| `QuestAccept` | 7 | An interaction shown when you're accepting a new quest. |
| `ProgressTab` | 8 | Honestly, this doesn't seem consistent to me. It is used to give you choices in the Cryptarch as well as some reward prompts by the Eververse vendor. I'll have to look into that further at some point. |
| `End` | 9 | These seem even less consistent. I don't know what these are. |
| `Start` | 10 | Also seem inconsistent. I also don't know what these are offhand. |

#### Destiny.Definitions.DestinyVendorInventoryFlyoutDefinition

**Type:** object

The definition for an "inventory flyout": a UI screen where we show you part of an otherwise hidden vendor inventory: like the Vault inventory buckets.

| Property | Type | Description |
| --- | --- | --- |
| `lockedDescription` | string | If the flyout is locked, this is the reason why. |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | The title and other common properties of the flyout. |
| `buckets` | array&lt;Destiny.Definitions.DestinyVendorInventoryFlyoutBucketDefinition&gt; | A list of inventory buckets and other metadata to show on the screen. |
| `flyoutId` | uint32 | An identifier for the flyout, in case anything else needs to refer to them. |
| `suppressNewness` | boolean | If this is true, don't show any of the glistening "this is a new item" UI elements, like we show on the inventory items themselves in in-game UI. |
| `equipmentSlotHash` | uint32? | If this flyout is meant to show you the contents of the player's equipment slot, this is the slot to show. |

#### Destiny.Definitions.DestinyVendorInventoryFlyoutBucketDefinition

**Type:** object

Information about a single inventory bucket in a vendor flyout UI and how it is shown.

| Property | Type | Description |
| --- | --- | --- |
| `collapsible` | boolean | If true, the inventory bucket should be able to be collapsed visually. |
| `inventoryBucketHash` | uint32 → DestinyInventoryBucketDefinition | The inventory bucket whose contents should be shown. |
| `sortItemsBy` | int32 | The methodology to use for sorting items from the flyout. |

#### Destiny.DestinyItemSortTypeEnumeration

**Enum** (`int32`)

Determines how items are sorted in an inventory bucket.

| Value | # | Description |
| --- | --- | --- |
| `ItemId` | 0 | — |
| `Timestamp` | 1 | — |
| `StackSize` | 2 | — |

#### Destiny.Definitions.DestinyVendorItemDefinition

**Type:** object

This represents an item being sold by the vendor.

| Property | Type | Description |
| --- | --- | --- |
| `vendorItemIndex` | int32 | The index into the DestinyVendorDefinition.saleList. This is what we use to refer to items being sold throughout live and definition data. |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier of the item being sold (DestinyInventoryItemDefinition). Note that a vendor can sell the same item in multiple ways, so don't assume that itemHash is a unique identifier for this entity. |
| `quantity` | int32 | The amount you will recieve of the item described in itemHash if you make the purchase. |
| `failureIndexes` | array&lt;int32&gt; | An list of indexes into the DestinyVendorDefinition.failureStrings array, indicating the possible failure strings that can be relevant for this item. |
| `currencies` | array&lt;Destiny.Definitions.DestinyVendorItemQuantity&gt; | This is a pre-compiled aggregation of item value and priceOverrideList, so that we have one place to check for what the purchaser must pay for the item. Use this instead of trying to piece together the price separately. The somewhat crappy part about this is that, now that item quantity overrides have dynamic modifiers, this will not necessarily be statically true. If you were using this instead of live data, switch to using live data. |
| `refundPolicy` | int32 | If this item can be refunded, this is the policy for what will be refundd, how, and in what time period. |
| `refundTimeLimit` | int32 | The amount of time before refundability of the newly purchased item will expire. |
| `creationLevels` | array&lt;Destiny.Definitions.DestinyItemCreationEntryLevelDefinition&gt; | The Default level at which the item will spawn. Almost always driven by an adjusto these days. Ideally should be singular. It's a long story how this ended up as a list, but there is always either going to be 0:1 of these entities. |
| `displayCategoryIndex` | int32 | This is an index specifically into the display category, as opposed to the server-side Categories (which do not need to match or pair with each other in any way: server side categories are really just structures for common validation. Display Category will let us more easily categorize items visually) |
| `categoryIndex` | int32 | The index into the DestinyVendorDefinition.categories array, so you can find the category associated with this item. |
| `originalCategoryIndex` | int32 | Same as above, but for the original category indexes. |
| `minimumLevel` | int32 | The minimum character level at which this item is available for sale. |
| `maximumLevel` | int32 | The maximum character level at which this item is available for sale. |
| `action` | Destiny.Definitions.DestinyVendorSaleItemActionBlockDefinition | The action to be performed when purchasing the item, if it's not just "buy". |
| `displayCategory` | string | The string identifier for the category selling this item. |
| `inventoryBucketHash` | uint32 → DestinyInventoryBucketDefinition | The inventory bucket into which this item will be placed upon purchase. |
| `visibilityScope` | int32 | The most restrictive scope that determines whether the item is available in the Vendor's inventory. See DestinyGatingScope's documentation for more information. This can be determined by Unlock gating, or by whether or not the item has purchase level requirements (minimumLevel and maximumLevel properties). |
| `purchasableScope` | int32 | Similar to visibilityScope, it represents the most restrictive scope that determines whether the item can be purchased. It will at least be as restrictive as visibilityScope, but could be more restrictive if the item has additional purchase requirements beyond whether it is merely visible or not. See DestinyGatingScope's documentation for more information. |
| `exclusivity` | int32 | If this item can only be purchased by a given platform, this indicates the platform to which it is restricted. |
| `isOffer` | boolean? | If this sale can only be performed as the result of an offer check, this is true. |
| `isCrm` | boolean? | If this sale can only be performed as the result of receiving a CRM offer, this is true. |
| `sortValue` | int32 | *if* the category this item is in supports non-default sorting, this value should represent the sorting value to use, pre-processed and ready to go. |
| `expirationTooltip` | string | If this item can expire, this is the tooltip message to show with its expiration info. |
| `redirectToSaleIndexes` | array&lt;int32&gt; | If this is populated, the purchase of this item should redirect to purchasing these other items instead. |
| `socketOverrides` | array&lt;Destiny.Definitions.DestinyVendorItemSocketOverride&gt; | — |
| `unpurchasable` | boolean? | If true, this item is some sort of dummy sale item that cannot actually be purchased. It may be a display only item, or some fluff left by a content designer for testing purposes, or something that got disabled because it was a terrible idea. You get the picture. We won't know *why* it can't be purchased, only that it can't be. Sorry. This is also only whether it's unpurchasable as a static property according to game content. There are other reasons why an item may or may not be purchasable at runtime, so even if this isn't set to True you should trust the runtime value for this sale item over the static definition if this is unset. |

#### Destiny.Definitions.DestinyVendorItemQuantity

**Type:** object

In addition to item quantity information for vendor prices, this also has any optional information that may exist about how the item's quantity can be modified. (unfortunately not information that is able to be read outside of the BNet servers, but it's there)

| Property | Type | Description |
| --- | --- | --- |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier for the item in question. Use it to look up the item's DestinyInventoryItemDefinition. |
| `itemInstanceId` | int64? | If this quantity is referring to a specific instance of an item, this will have the item's instance ID. Normally, this will be null. |
| `quantity` | int32 | The amount of the item needed/available depending on the context of where DestinyItemQuantity is being used. |
| `hasConditionalVisibility` | boolean | Indicates that this item quantity may be conditionally shown or hidden, based on various sources of state. For example: server flags, account state, or character progress. |

#### Destiny.DestinyVendorItemRefundPolicyEnumeration

**Enum** (`int32`)

The action that happens when the user attempts to refund an item.

| Value | # | Description |
| --- | --- | --- |
| `NotRefundable` | 0 | — |
| `DeletesItem` | 1 | — |
| `RevokesLicense` | 2 | — |

#### Destiny.Definitions.DestinyItemCreationEntryLevelDefinition

**Type:** object

An overly complicated wrapper for the item level at which the item should spawn.

| Property | Type | Description |
| --- | --- | --- |
| `level` | int32 | — |

#### Destiny.Definitions.DestinyVendorSaleItemActionBlockDefinition

**Type:** object

Not terribly useful, some basic cooldown interaction info.

| Property | Type | Description |
| --- | --- | --- |
| `executeSeconds` | float | — |
| `isPositive` | boolean | — |

#### Destiny.DestinyGatingScopeEnumeration

**Enum** (`int32`)

This enumeration represents the most restrictive type of gating that is being performed by an entity. This is useful as a shortcut to avoid a lot of lookups when determining whether the gating on an Entity applies to everyone equally, or to their specific Profile or Character states. None = There is no gating on this item. Global = The gating on this item is based entirely on global game state. It will be gated the same for everyone. Clan = The gating on this item is at the Clan level. For instance, if you're gated by Clan level this will be the case. Profile = The gating includes Profile-specific checks, but not on the Profile's characters. An example of this might be when you acquire an Emblem: the Emblem will be available in your Kiosk for all characters in your Profile from that point onward. Character = The gating includes Character-specific checks, including character level restrictions. An example of this might be an item that you can't purchase from a Vendor until you reach a specific Character Level. Item = The gating includes item-specific checks. For BNet, this generally implies that we'll show this data only on a character level or deeper. AssumedWorstCase = The unlocks and checks being used for this calculation are of an unknown type and are used for unknown purposes. For instance, if some great person decided that an unlock value should be globally scoped, but then the game changes it using character-specific data in a way that BNet doesn't know about. Because of the open-ended potential for this to occur, many unlock checks for "globally" scoped unlock data may be assumed as the worst case unless it has been specifically whitelisted as otherwise. That sucks, but them's the breaks.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Global` | 1 | — |
| `Clan` | 2 | — |
| `Profile` | 3 | — |
| `Character` | 4 | — |
| `Item` | 5 | — |
| `AssumedWorstCase` | 6 | — |

#### Destiny.Definitions.DestinyVendorItemSocketOverride

**Type:** object

The information for how the vendor purchase should override a given socket with custom plug data.

| Property | Type | Description |
| --- | --- | --- |
| `singleItemHash` | uint32 → DestinyInventoryItemDefinition? | If this is populated, the socket will be overridden with a specific plug. If this isn't populated, it's being overridden by something more complicated that is only known by the Game Server and God, which means we can't tell you in advance what it'll be. |
| `randomizedOptionsCount` | int32 | If this is greater than -1, the number of randomized plugs on this socket will be set to this quantity instead of whatever it's set to by default. |
| `socketTypeHash` | uint32 → DestinySocketTypeDefinition | This appears to be used to select which socket ultimately gets the override defined here. |

#### Destiny.Definitions.DestinyVendorServiceDefinition

**Type:** object

When a vendor provides services, this is the localized name of those services.

| Property | Type | Description |
| --- | --- | --- |
| `name` | string | The localized name of a service provided. |

#### Destiny.Definitions.DestinyVendorAcceptedItemDefinition

**Type:** object

If you ever wondered how the Vault works, here it is. The Vault is merely a set of inventory buckets that exist on your Profile/Account level. When you transfer items in the Vault, the game is using the Vault Vendor's DestinyVendorAcceptedItemDefinitions to see where the appropriate destination bucket is for the source bucket from whence your item is moving. If it finds such an entry, it transfers the item to the other bucket. The mechanics for Postmaster works similarly, which is also a vendor. All driven by Accepted Items.

| Property | Type | Description |
| --- | --- | --- |
| `acceptedInventoryBucketHash` | uint32 → DestinyInventoryBucketDefinition | The "source" bucket for a transfer. When a user wants to transfer an item, the appropriate DestinyVendorDefinition's acceptedItems property is evaluated, looking for an entry where acceptedInventoryBucketHash matches the bucket that the item being transferred is currently located. If it exists, the item will be transferred into whatever bucket is defined by destinationInventoryBucketHash. |
| `destinationInventoryBucketHash` | uint32 → DestinyInventoryBucketDefinition | This is the bucket where the item being transferred will be put, given that it was being transferred *from* the bucket defined in acceptedInventoryBucketHash. |

#### Destiny.Definitions.Vendors.DestinyVendorLocationDefinition

**Type:** object

These definitions represent vendors' locations and relevant display information at different times in the game.

| Property | Type | Description |
| --- | --- | --- |
| `destinationHash` | uint32 → DestinyDestinationDefinition | The hash identifier for a Destination at which this vendor may be located. Each destination where a Vendor may exist will only ever have a single entry. |
| `backgroundImagePath` | string | The relative path to the background image representing this Vendor at this location, for use in a banner. |

#### Destiny.Definitions.DestinyDestinationDefinition

**Object** · *(Manifest definition, table `Destinations`)*

On to one of the more confusing subjects of the API. What is a Destination, and what is the relationship between it, Activities, Locations, and Places? A "Destination" is a specific region/city/area of a larger "Place". For instance, a Place might be Earth where a Destination might be Bellevue, Washington. (Please, pick a more interesting destination if you come to visit Earth).

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `placeHash` | uint32 → DestinyPlaceDefinition | The place that "owns" this Destination. Use this hash to look up the DestinyPlaceDefinition. |
| `defaultFreeroamActivityHash` | uint32 → DestinyActivityDefinition | If this Destination has a default Free-Roam activity, this is the hash for that Activity. Use it to look up the DestinyActivityDefintion. |
| `activityGraphEntries` | array&lt;Destiny.Definitions.DestinyActivityGraphListEntryDefinition&gt; | If the Destination has default Activity Graphs (i.e. "Map") that should be shown in the director, this is the list of those Graphs. At most, only one should be active at any given time for a Destination: these would represent, for example, different variants on a Map if the Destination is changing on a macro level based on game state. |
| `bubbleSettings` | array&lt;Destiny.Definitions.DestinyDestinationBubbleSettingDefinition&gt; | A Destination may have many "Bubbles" zones with human readable properties. We don't get as much info as I'd like about them - I'd love to return info like where on the map they are located - but at least this gives you the name of those bubbles. bubbleSettings and bubbles both have the identical number of entries, and you should match up their indexes to provide matching bubble and bubbleSettings data. DEPRECATED - Just use bubbles, it now has this data. |
| `bubbles` | array&lt;Destiny.Definitions.DestinyBubbleDefinition&gt; | This provides the unique identifiers for every bubble in the destination (only guaranteed unique within the destination), and any intrinsic properties of the bubble. bubbleSettings and bubbles both have the identical number of entries, and you should match up their indexes to provide matching bubble and bubbleSettings data. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyActivityGraphListEntryDefinition

**Type:** object

Destinations and Activities may have default Activity Graphs that should be shown when you bring up the Director and are playing in either. This contract defines the graph referred to and the gating for when it is relevant.

| Property | Type | Description |
| --- | --- | --- |
| `activityGraphHash` | uint32 → DestinyActivityGraphDefinition | The hash identifier of the DestinyActivityGraphDefinition that should be shown when opening the director. |

#### Destiny.Definitions.Director.DestinyActivityGraphDefinition

**Object** · *(Manifest definition, table `ActivityGraphs`)*

Represents a Map View in the director: be them overview views, destination views, or other. They have nodes which map to activities, and other various visual elements that we (or others) may or may not be able to use. Activity graphs, most importantly, have nodes which can have activities in various states of playability. Unfortunately, activity graphs are combined at runtime with Game UI-only assets such as fragments of map images, various in-game special effects, decals etc... that we don't get in these definitions. If we end up having time, we may end up trying to manually populate those here: but the last time we tried that, before the lead-up to D1, it proved to be unmaintainable as the game's content changed. So don't bet the farm on us providing that content in this definition.

| Property | Type | Description |
| --- | --- | --- |
| `nodes` | array&lt;Destiny.Definitions.Director.DestinyActivityGraphNodeDefinition&gt; | These represent the visual "nodes" on the map's view. These are the activities you can click on in the map. |
| `artElements` | array&lt;Destiny.Definitions.Director.DestinyActivityGraphArtElementDefinition&gt; | Represents one-off/special UI elements that appear on the map. |
| `connections` | array&lt;Destiny.Definitions.Director.DestinyActivityGraphConnectionDefinition&gt; | Represents connections between graph nodes. However, it lacks context that we'd need to make good use of it. |
| `displayObjectives` | array&lt;Destiny.Definitions.Director.DestinyActivityGraphDisplayObjectiveDefinition&gt; | Objectives can display on maps, and this is supposedly metadata for that. I have not had the time to analyze the details of what is useful within however: we could be missing important data to make this work. Expect this property to be expanded on later if possible. |
| `displayProgressions` | array&lt;Destiny.Definitions.Director.DestinyActivityGraphDisplayProgressionDefinition&gt; | Progressions can also display on maps, but similarly to displayObjectives we appear to lack some required information and context right now. We will have to look into it later and add more data if possible. |
| `linkedGraphs` | array&lt;Destiny.Definitions.Director.DestinyLinkedGraphDefinition&gt; | Represents links between this Activity Graph and other ones. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Director.DestinyActivityGraphNodeDefinition

**Type:** object

This is the position and other data related to nodes in the activity graph that you can click to launch activities. An Activity Graph node will only have one active Activity at a time, which will determine the activity to be launched (and, unless overrideDisplay information is provided, will also determine the tooltip and other UI related to the node)

| Property | Type | Description |
| --- | --- | --- |
| `nodeId` | uint32 | An identifier for the Activity Graph Node, only guaranteed to be unique within its parent Activity Graph. |
| `overrideDisplay` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | The node *may* have display properties that override the active Activity's display properties. |
| `position` | Destiny.Definitions.Common.DestinyPositionDefinition | The position on the map for this node. |
| `featuringStates` | array&lt;Destiny.Definitions.Director.DestinyActivityGraphNodeFeaturingStateDefinition&gt; | The node may have various visual accents placed on it, or styles applied. These are the list of possible styles that the Node can have. The game iterates through each, looking for the first one that passes a check of the required game/character/account state in order to show that style, and then renders the node in that style. |
| `activities` | array&lt;Destiny.Definitions.Director.DestinyActivityGraphNodeActivityDefinition&gt; | The node may have various possible activities that could be active for it, however only one may be active at a time. See the DestinyActivityGraphNodeActivityDefinition for details. |
| `states` | array&lt;Destiny.Definitions.Director.DestinyActivityGraphNodeStateEntry&gt; | Represents possible states that the graph node can be in. These are combined with some checking that happens in the game client and server to determine which state is actually active at any given time. |

#### Destiny.Definitions.Common.DestinyPositionDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `x` | int32 | — |
| `y` | int32 | — |
| `z` | int32 | — |

#### Destiny.Definitions.Director.DestinyActivityGraphNodeFeaturingStateDefinition

**Type:** object

Nodes can have different visual states. This object represents a single visual state ("highlight type") that a node can be in, and the unlock expression condition to determine whether it should be set.

| Property | Type | Description |
| --- | --- | --- |
| `highlightType` | int32 | The node can be highlighted in a variety of ways - the game iterates through these and finds the first FeaturingState that is valid at the present moment given the Game, Account, and Character state, and renders the node in that state. See the ActivityGraphNodeHighlightType enum for possible values. |

#### Destiny.ActivityGraphNodeHighlightTypeEnumeration

**Enum** (`int32`)

The various known UI styles in which an item can be highlighted. It'll be up to you to determine what you want to show based on this highlighting, BNet doesn't have any assets that correspond to these states. And yeah, RiseOfIron and Comet have their own special highlight states. Don't ask me, I can't imagine they're still used.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Normal` | 1 | — |
| `Hyper` | 2 | — |
| `Comet` | 3 | — |
| `RiseOfIron` | 4 | — |

#### Destiny.Definitions.Director.DestinyActivityGraphNodeActivityDefinition

**Type:** object

The actual activity to be redirected to when you click on the node. Note that a node can have many Activities attached to it: but only one will be active at any given time. The list of Node Activities will be traversed, and the first one found to be active will be displayed. This way, a node can layer multiple variants of an activity on top of each other. For instance, one node can control the weekly Crucible Playlist. There are multiple possible playlists, but only one is active for the week.

| Property | Type | Description |
| --- | --- | --- |
| `nodeActivityId` | uint32 | An identifier for this node activity. It is only guaranteed to be unique within the Activity Graph. |
| `activityHash` | uint32 → DestinyActivityDefinition | The activity that will be activated if the user clicks on this node. Controls all activity-related information displayed on the node if it is active (the text shown in the tooltip etc) |

#### Destiny.Definitions.DestinyActivityDefinition

**Object** · *(Manifest definition, table `Activities`)*

The static data about Activities in Destiny 2. Note that an Activity must be combined with an ActivityMode to know - from a Gameplay perspective - what the user is "Playing". In most PvE activities, this is fairly straightforward. A Story Activity can only be played in the Story Activity Mode. However, in PvP activities, the Activity alone only tells you the map being played, or the Playlist that the user chose to enter. You'll need to know the Activity Mode they're playing to know that they're playing Mode X on Map Y. Activity Definitions tell a great deal of information about what *could* be relevant to a user: what rewards they can earn, what challenges could be performed, what modifiers could be applied. To figure out which of these properties is actually live, you'll need to combine the definition with "Live" data from one of the Destiny endpoints. Activities also have Activity Types, but unfortunately in Destiny 2 these are even less reliable of a source of information than they were in Destiny 1. I will be looking into ways to provide more reliable sources for type information as time goes on, but for now we're going to have to deal with the limitations. See DestinyActivityTypeDefinition for more information.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | The title, subtitle, and icon for the activity. We do a little post-processing on this to try and account for Activities where the designers have left this data too minimal to determine what activity is actually being played. |
| `originalDisplayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | The unadulterated form of the display properties, as they ought to be shown in the Director (if the activity appears in the director). |
| `selectionScreenDisplayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | The title, subtitle, and icon for the activity as determined by Selection Screen data, if there is any for this activity. There won't be data in this field if the activity is never shown in a selection/options screen. |
| `releaseIcon` | string | If the activity has an icon associated with a specific release (such as a DLC), this is the path to that release's icon. |
| `releaseTime` | int32 | If the activity will not be visible until a specific and known time, this will be the seconds since the Epoch when it will become visible. |
| `activityLightLevel` | int32 | The recommended light level for this activity. |
| `destinationHash` | uint32 → DestinyDestinationDefinition | The hash identifier for the Destination on which this Activity is played. Use it to look up the DestinyDestinationDefinition for human readable info about the destination. A Destination can be thought of as a more specific location than a "Place". For instance, if the "Place" is Earth, the "Destination" would be a specific city or region on Earth. |
| `placeHash` | uint32 → DestinyPlaceDefinition | The hash identifier for the "Place" on which this Activity is played. Use it to look up the DestinyPlaceDefinition for human readable info about the Place. A Place is the largest-scoped concept for location information. For instance, if the "Place" is Earth, the "Destination" would be a specific city or region on Earth. |
| `activityTypeHash` | uint32 → DestinyActivityTypeDefinition | The hash identifier for the Activity Type of this Activity. You may use it to look up the DestinyActivityTypeDefinition for human readable info, but be forewarned: Playlists and many PVP Map Activities will map to generic Activity Types. You'll have to use your knowledge of the Activity Mode being played to get more specific information about what the user is playing. |
| `tier` | int32 | The difficulty tier of the activity. |
| `pgcrImage` | string | When Activities are completed, we generate a "Post-Game Carnage Report", or PGCR, with details about what happened in that activity (how many kills someone got, which team won, etc...) We use this image as the background when displaying PGCR information, and often use it when we refer to the Activity in general. |
| `rewards` | array&lt;Destiny.Definitions.DestinyActivityRewardDefinition&gt; | The expected possible rewards for the activity. These rewards may or may not be accessible for an individual player based on their character state, the account state, and even the game's state overall. But it is a useful reference for possible rewards you can earn in the activity. These match up to rewards displayed when you hover over the Activity in the in-game Director, and often refer to Placeholder or "Dummy" items: items that tell you what you can earn in vague terms rather than what you'll specifically be earning (partly because the game doesn't even know what you'll earn specifically until you roll for it at the end) |
| `modifiers` | array&lt;Destiny.Definitions.DestinyActivityModifierReferenceDefinition&gt; | Activities can have Modifiers, as defined in DestinyActivityModifierDefinition. These are references to the modifiers that *can* be applied to that activity, along with data that we use to determine if that modifier is actually active at any given point in time. |
| `isPlaylist` | boolean | If True, this Activity is actually a Playlist that refers to multiple possible specific Activities and Activity Modes. For instance, a Crucible Playlist may have references to multiple Activities (Maps) with multiple Activity Modes (specific PvP gameplay modes). If this is true, refer to the playlistItems property for the specific entries in the playlist. |
| `challenges` | array&lt;Destiny.Definitions.DestinyActivityChallengeDefinition&gt; | An activity can have many Challenges, of which any subset of them may be active for play at any given period of time. This gives the information about the challenges and data that we use to understand when they're active and what rewards they provide. Sadly, at the moment there's no central definition for challenges: much like "Skulls" were in Destiny 1, these are defined on individual activities and there can be many duplicates/near duplicates across the Destiny 2 ecosystem. I have it in mind to centralize these in a future revision of the API, but we are out of time. |
| `optionalUnlockStrings` | array&lt;Destiny.Definitions.DestinyActivityUnlockStringDefinition&gt; | If there are status strings related to the activity and based on internal state of the game, account, or character, then this will be the definition of those strings and the states needed in order for the strings to be shown. |
| `activityFamilyHashes` | array&lt;uint32&gt; → DestinyActivityFamilyDefinition | — |
| `traitHashes` | array&lt;uint32&gt; → DestinyTraitDefinition | — |
| `requirements` | Destiny.Definitions.DestinyActivityRequirementsBlock | — |
| `difficultyTierCollectionHash` | uint32 → DestinyActivityDifficultyTierCollectionDefinition? | — |
| `selectableSkullCollectionHashes` | array&lt;uint32&gt; → DestinyActivitySelectableSkullCollectionDefinition | — |
| `selectableSkullCollections` | array&lt;Destiny.Definitions.DestinyActivitySelectableSkullCollections&gt; → DestinyActivityDifficultyTierCollectionDefinition | — |
| `playlistItems` | array&lt;Destiny.Definitions.DestinyActivityPlaylistItemDefinition&gt; | Represents all of the possible activities that could be played in the Playlist, along with information that we can use to determine if they are active at the present time. |
| `activityGraphList` | array&lt;Destiny.Definitions.DestinyActivityGraphListEntryDefinition&gt; | Unfortunately, in practice this is almost never populated. In theory, this is supposed to tell which Activity Graph to show if you bring up the director while in this activity. |
| `matchmaking` | Destiny.Definitions.DestinyActivityMatchmakingBlockDefinition | This block of data provides information about the Activity's matchmaking attributes: how many people can join and such. |
| `guidedGame` | Destiny.Definitions.DestinyActivityGuidedBlockDefinition | This block of data, if it exists, provides information about the guided game experience and restrictions for this activity. If it doesn't exist, the game is not able to be played as a guided game. |
| `directActivityModeHash` | uint32 → DestinyActivityModeDefinition? | If this activity had an activity mode directly defined on it, this will be the hash of that mode. |
| `directActivityModeType` | int32? | If the activity had an activity mode directly defined on it, this will be the enum value of that mode. |
| `loadouts` | array&lt;Destiny.Definitions.DestinyActivityLoadoutRequirementSet&gt; | The set of all possible loadout requirements that could be active for this activity. Only one will be active at any given time, and you can discover which one through activity-associated data such as Milestones that have activity info on them. |
| `activityModeHashes` | array&lt;uint32&gt; → DestinyActivityModeDefinition | The hash identifiers for Activity Modes relevant to this activity. Note that if this is a playlist, the specific playlist entry chosen will determine the actual activity modes that end up being relevant. |
| `activityModeTypes` | array&lt;int32&gt; | The activity modes - if any - in enum form. Because we can't seem to escape the enums. |
| `isPvP` | boolean | If true, this activity is a PVP activity or playlist. |
| `insertionPoints` | array&lt;Destiny.Definitions.DestinyActivityInsertionPointDefinition&gt; | The list of phases or points of entry into an activity, along with information we can use to determine their gating and availability. |
| `activityLocationMappings` | array&lt;Destiny.Constants.DestinyEnvironmentLocationMapping&gt; | A list of location mappings that are affected by this activity. Pulled out of DestinyLocationDefinitions for our/your lookup convenience. |
| `curatorBlockDefinition` | Destiny.Definitions.DestinyActivityCuratorBlockDefinition | Additional data used for display in the in-game Portal screen |
| `durationEstimate` | Destiny.Definitions.DestinyActivityDurationEstimate | Optional estimated duration, shown on the Portal tiles |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyActivityRewardDefinition

**Type:** object

Activities can refer to one or more sets of tooltip-friendly reward data. These are the definitions for those tooltip friendly rewards.

| Property | Type | Description |
| --- | --- | --- |
| `rewardText` | string | The header for the reward set, if any. |
| `rewardItems` | array&lt;Destiny.DestinyItemQuantity&gt; | The "Items provided" in the reward. This is almost always a pointer to a DestinyInventoryItemDefintion for an item that you can't actually earn in-game, but that has name/description/icon information for the vague concept of the rewards you will receive. This is because the actual reward generation is non-deterministic and extremely complicated, so the best the game can do is tell you what you'll get in vague terms. And so too shall we. Interesting trivia: you actually *do* earn these items when you complete the activity. They go into a single-slot bucket on your profile, which is how you see the pop-ups of these rewards when you complete an activity that match these "dummy" items. You can even see them if you look at the last one you earned in your profile-level inventory through the BNet API! Who said reading documentation is a waste of time? |

#### Destiny.Definitions.DestinyActivityModifierReferenceDefinition

**Type:** object

A reference to an Activity Modifier from another entity, such as an Activity (for now, just Activities). This defines some

| Property | Type | Description |
| --- | --- | --- |
| `activityModifierHash` | uint32 → DestinyActivityModifierDefinition | The hash identifier for the DestinyActivityModifierDefinition referenced by this activity. |

#### Destiny.Definitions.ActivityModifiers.DestinyActivityModifierDefinition

**Object** · *(Manifest definition, table `ActivityModifiers`)*

Modifiers - in Destiny 1, these were referred to as "Skulls" - are changes that can be applied to an Activity.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `displayInNavMode` | boolean | — |
| `displayInActivitySelection` | boolean | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyActivityChallengeDefinition

**Type:** object

Represents a reference to a Challenge, which for now is just an Objective.

| Property | Type | Description |
| --- | --- | --- |
| `objectiveHash` | uint32 → DestinyObjectiveDefinition | The hash for the Objective that matches this challenge. Use it to look up the DestinyObjectiveDefinition. |
| `dummyRewards` | array&lt;Destiny.DestinyItemQuantity&gt; | The rewards as they're represented in the UI. Note that they generally link to "dummy" items that give a summary of rewards rather than direct, real items themselves. If the quantity is 0, don't show the quantity. |

#### Destiny.Definitions.DestinyObjectiveDefinition

**Object** · *(Manifest definition, table `Objectives`)*

Defines an "Objective". An objective is a specific task you should accomplish in the game. These are referred to by: - Quest Steps (which are DestinyInventoryItemDefinition entities with Objectives) - Challenges (which are Objectives defined on an DestinyActivityDefintion) - Milestones (which refer to Objectives that are defined on both Quest Steps and Activities) - Anything else that the designers decide to do later. Objectives have progress, a notion of having been Completed, human readable data describing the task to be accomplished, and a lot of optional tack-on data that can enhance the information provided about the task.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | Ideally, this should tell you what your task is. I'm not going to lie to you though. Sometimes this doesn't have useful information at all. Which sucks, but there's nothing either of us can do about it. |
| `completionValue` | int32 | The value that the unlock value defined in unlockValueHash must reach in order for the objective to be considered Completed. Used in calculating progress and completion status. |
| `scope` | int32 | A shortcut for determining the most restrictive gating that this Objective is set to use. This includes both the dynamic determination of progress and of completion values. See the DestinyGatingScope enum's documentation for more details. |
| `locationHash` | uint32 → DestinyLocationDefinition | OPTIONAL: a hash identifier for the location at which this objective must be accomplished, if there is a location defined. Look up the DestinyLocationDefinition for this hash for that additional location info. |
| `allowNegativeValue` | boolean | If true, the value is allowed to go negative. |
| `allowValueChangeWhenCompleted` | boolean | If true, you can effectively "un-complete" this objective if you lose progress after crossing the completion threshold. If False, once you complete the task it will remain completed forever by locking the value. |
| `isCountingDownward` | boolean | If true, completion means having an unlock value less than or equal to the completionValue. If False, completion means having an unlock value greater than or equal to the completionValue. |
| `valueStyle` | int32 | The UI style applied to the objective. It's an enum, take a look at DestinyUnlockValueUIStyle for details of the possible styles. Use this info as you wish to customize your UI. DEPRECATED: This is no longer populated by Destiny 2 game content. Please use inProgressValueStyle and completedValueStyle instead. |
| `progressDescription` | string | Text to describe the progress bar. |
| `perks` | Destiny.Definitions.DestinyObjectivePerkEntryDefinition | If this objective enables Perks intrinsically, the conditions for that enabling are defined here. |
| `stats` | Destiny.Definitions.DestinyObjectiveStatEntryDefinition | If this objective enables modifications on a player's stats intrinsically, the conditions are defined here. |
| `minimumVisibilityThreshold` | int32 | If nonzero, this is the minimum value at which the objective's progression should be shown. Otherwise, don't show it yet. |
| `allowOvercompletion` | boolean | If True, the progress will continue even beyond the point where the objective met its minimum completion requirements. Your UI will have to accommodate it. |
| `showValueOnComplete` | boolean | If True, you should continue showing the progression value in the UI after it's complete. I mean, we already do that in BNet anyways, but if you want to be better behaved than us you could honor this flag. |
| `completedValueStyle` | int32 | The style to use when the objective is completed. |
| `inProgressValueStyle` | int32 | The style to use when the objective is still in progress. |
| `uiLabel` | string | Objectives can have arbitrary UI-defined identifiers that define the style applied to objectives. For convenience, known UI labels will be defined in the uiStyle enum value. |
| `uiStyle` | int32 | If the objective has a known UI label value, this property will represent it. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.DestinyUnlockValueUIStyleEnumeration

**Enum** (`int32`)

If you're showing an unlock value in the UI, this is the format in which it should be shown. You'll have to build your own algorithms on the client side to determine how best to render these options.

| Value | # | Description |
| --- | --- | --- |
| `Automatic` | 0 | Generally, Automatic means "Just show the number" |
| `Fraction` | 1 | Show the number as a fractional value. For this to make sense, the value being displayed should have a comparable upper bound, like the progress to the next level of a Progression. |
| `Checkbox` | 2 | Show the number as a checkbox. 0 Will mean unchecked, any other value will mean checked. |
| `Percentage` | 3 | Show the number as a percentage. For this to make sense, the value being displayed should have a comparable upper bound, like the progress to the next level of a Progression. |
| `DateTime` | 4 | Show the number as a date and time. The number will be the number of seconds since the Unix Epoch (January 1st, 1970 at midnight UTC). It'll be up to you to convert this into a date and time format understandable to the user in their time zone. |
| `FractionFloat` | 5 | Show the number as a floating point value that represents a fraction, where 0 is min and 1 is max. For this to make sense, the value being displayed should have a comparable upper bound, like the progress to the next level of a Progression. |
| `Integer` | 6 | Show the number as a straight-up integer. |
| `TimeDuration` | 7 | Show the number as a time duration. The value will be returned as seconds. |
| `Hidden` | 8 | Don't bother showing the value at all, it's not easily human-interpretable, and used for some internal purpose. |
| `Multiplier` | 9 | Example: "1.5x" |
| `GreenPips` | 10 | Show the value as a series of green pips, like the wins in a Trials of Osiris score card. |
| `RedPips` | 11 | Show the value as a series of red pips, like the losses in a Trials of Osiris score card. |
| `ExplicitPercentage` | 12 | Show the value as a percentage. For example: "51%" - Does no division, only appends '%' |
| `RawFloat` | 13 | Show the value as a floating-point number. For example: "4.52" NOTE: Passed along from Investment as whole number with last two digits as decimal values (452 -> 4.52) |
| `LevelAndReward` | 14 | Show the value as a level and a reward. |

#### Destiny.Definitions.DestinyObjectivePerkEntryDefinition

**Type:** object

Defines the conditions under which an intrinsic perk is applied while participating in an Objective. These perks will generally not be benefit-granting perks, but rather a perk that modifies gameplay in some interesting way.

| Property | Type | Description |
| --- | --- | --- |
| `perkHash` | uint32 → DestinySandboxPerkDefinition | The hash identifier of the DestinySandboxPerkDefinition that will be applied to the character. |
| `style` | int32 | An enumeration indicating whether it will be applied as long as the Objective is active, when it's completed, or until it's completed. |

#### Destiny.DestinyObjectiveGrantStyleEnumeration

**Enum** (`int32`)

Some Objectives provide perks, generally as part of providing some kind of interesting modifier for a Challenge or Quest. This indicates when the Perk is granted.

| Value | # | Description |
| --- | --- | --- |
| `WhenIncomplete` | 0 | — |
| `WhenComplete` | 1 | — |
| `Always` | 2 | — |

#### Destiny.Definitions.DestinyObjectiveStatEntryDefinition

**Type:** object

Defines the conditions under which stat modifications will be applied to a Character while participating in an objective.

| Property | Type | Description |
| --- | --- | --- |
| `stat` | Destiny.Definitions.DestinyItemInvestmentStatDefinition | The stat being modified, and the value used. |
| `style` | int32 | Whether it will be applied as long as the objective is active, when it's completed, or until it's completed. |

#### Destiny.Definitions.DestinyItemInvestmentStatDefinition

**Type:** object

Represents a "raw" investment stat, before calculated stats are calculated and before any DestinyStatGroupDefinition is applied to transform the stat into something closer to what you see in- game. Because these won't match what you see in-game, consider carefully whether you really want to use these stats. I have left them in case someone can do something useful or interesting with the pre- processed statistics.

| Property | Type | Description |
| --- | --- | --- |
| `statTypeHash` | uint32 → DestinyStatDefinition | The hash identifier for the DestinyStatDefinition defining this stat. |
| `value` | int32 | The raw "Investment" value for the stat, before transformations are performed to turn this raw stat into stats that are displayed in the game UI. |
| `isConditionallyActive` | boolean | If this is true, the stat will only be applied on the item in certain game state conditions, and we can't know statically whether or not this stat will be applied. Check the "live" API data instead for whether this value is being applied on a specific instance of the item in question, and you can use this to decide whether you want to show the stat on the generic view of the item, or whether you want to show some kind of caveat or warning about the stat value being conditional on game state. |

#### Destiny.DestinyObjectiveUiStyleEnumeration

**Enum** (`int32`)

If the objective has a known UI label, this enumeration will represent it.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Highlighted` | 1 | — |
| `CraftingWeaponLevel` | 2 | — |
| `CraftingWeaponLevelProgress` | 3 | — |
| `CraftingWeaponTimestamp` | 4 | — |
| `CraftingMementos` | 5 | — |
| `CraftingMementoTitle` | 6 | — |
| `DiscoverableMystery0` | 7 | — |
| `DiscoverableMystery1` | 8 | — |
| `DiscoverableMystery2` | 9 | — |
| `DiscoverableMystery3` | 10 | — |
| `DiscoverableMystery4` | 11 | — |
| `DiscoverableExotic` | 12 | — |

#### Destiny.Definitions.DestinyLocationDefinition

**Object** · *(Manifest definition, table `Locations`)*

A "Location" is a sort of shortcut for referring to a specific combination of Activity, Destination, Place, and even Bubble or NavPoint within a space. Most of this data isn't intrinsically useful to us, but Objectives refer to locations, and through that we can at least infer the Activity, Destination, and Place being referred to by the Objective.

| Property | Type | Description |
| --- | --- | --- |
| `vendorHash` | uint32 → DestinyVendorDefinition | If the location has a Vendor on it, this is the hash identifier for that Vendor. Look them up with DestinyVendorDefinition. |
| `locationReleases` | array&lt;Destiny.Definitions.DestinyLocationReleaseDefinition&gt; | A Location may refer to different specific spots in the world based on the world's current state. This is a list of those potential spots, and the data we can use at runtime to determine which one of the spots is the currently valid one. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyLocationReleaseDefinition

**Type:** object

A specific "spot" referred to by a location. Only one of these can be active at a time for a given Location.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | Sadly, these don't appear to be populated anymore (ever?) |
| `smallTransparentIcon` | string | — |
| `mapIcon` | string | — |
| `largeTransparentIcon` | string | — |
| `spawnPoint` | uint32 | If we had map information, this spawnPoint would be interesting. But sadly, we don't have that info. |
| `destinationHash` | uint32 → DestinyDestinationDefinition | The Destination being pointed to by this location. |
| `activityHash` | uint32 → DestinyActivityDefinition | The Activity being pointed to by this location. |
| `activityGraphHash` | uint32 | The Activity Graph being pointed to by this location. |
| `activityGraphNodeHash` | uint32 | The Activity Graph Node being pointed to by this location. (Remember that Activity Graph Node hashes are only unique within an Activity Graph: so use the combination to find the node being spoken of) |
| `activityBubbleName` | uint32 | The Activity Bubble within the Destination. Look this up in the DestinyDestinationDefinition's bubbles and bubbleSettings properties. |
| `activityPathBundle` | uint32 | If we had map information, this would tell us something cool about the path this location wants you to take. I wish we had map information. |
| `activityPathDestination` | uint32 | If we had map information, this would tell us about path information related to destination on the map. Sad. Maybe you can do something cool with it. Go to town man. |
| `navPointType` | int32 | The type of Nav Point that this represents. See the enumeration for more info. |
| `worldPosition` | array&lt;int32&gt; | Looks like it should be the position on the map, but sadly it does not look populated... yet? |

#### Destiny.DestinyActivityNavPointTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Inactive` | 0 | — |
| `PrimaryObjective` | 1 | — |
| `SecondaryObjective` | 2 | — |
| `TravelObjective` | 3 | — |
| `PublicEventObjective` | 4 | — |
| `AmmoCache` | 5 | — |
| `PointTypeFlag` | 6 | — |
| `CapturePoint` | 7 | — |
| `DefensiveEncounter` | 8 | — |
| `GhostInteraction` | 9 | — |
| `KillAi` | 10 | — |
| `QuestItem` | 11 | — |
| `PatrolMission` | 12 | — |
| `Incoming` | 13 | — |
| `ArenaObjective` | 14 | — |
| `AutomationHint` | 15 | — |
| `TrackedQuest` | 16 | — |

#### Destiny.Definitions.DestinyActivityUnlockStringDefinition

**Type:** object

Represents a status string that could be conditionally displayed about an activity. Note that externally, you can only see the strings themselves. Internally we combine this information with server state to determine which strings should be shown.

| Property | Type | Description |
| --- | --- | --- |
| `displayString` | string | The string to be displayed if the conditions are met. |

#### Destiny.Definitions.DestinyActivityRequirementsBlock

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `leaderRequirementLabels` | array&lt;Destiny.Definitions.DestinyActivityRequirementLabel&gt; | If being a fireteam Leader in this activity is gated, this is the gate being checked. |
| `fireteamRequirementLabels` | array&lt;Destiny.Definitions.DestinyActivityRequirementLabel&gt; | If being a fireteam member in this activity is gated, this is the gate being checked. |

#### Destiny.Definitions.DestinyActivityRequirementLabel

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `displayString` | string | — |

#### Destiny.Definitions.DestinyActivitySelectableSkullCollections

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `selectableSkullCollectionHash` | uint32 | — |
| `minimumTierRank` | int32 | — |
| `maximumTierRank` | int32 | — |

#### Destiny.Definitions.DestinyActivityPlaylistItemDefinition

**Type:** object

If the activity is a playlist, this is the definition for a specific entry in the playlist: a single possible combination of Activity and Activity Mode that can be chosen.

| Property | Type | Description |
| --- | --- | --- |
| `activityHash` | uint32 → DestinyActivityDefinition | The hash identifier of the Activity that can be played. Use it to look up the DestinyActivityDefinition. |
| `directActivityModeHash` | uint32 → DestinyActivityModeDefinition? | If this playlist entry had an activity mode directly defined on it, this will be the hash of that mode. |
| `directActivityModeType` | int32? | If the playlist entry had an activity mode directly defined on it, this will be the enum value of that mode. |
| `activityModeHashes` | array&lt;uint32&gt; → DestinyActivityModeDefinition | The hash identifiers for Activity Modes relevant to this entry. |
| `activityModeTypes` | array&lt;int32&gt; | The activity modes - if any - in enum form. Because we can't seem to escape the enums. |

#### Destiny.HistoricalStats.Definitions.DestinyActivityModeTypeEnumeration

**Enum** (`int32`)

For historical reasons, this list will have both D1 and D2-relevant Activity Modes in it. Please don't take this to mean that some D1-only feature is coming back!

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Story` | 2 | — |
| `Strike` | 3 | — |
| `Raid` | 4 | — |
| `AllPvP` | 5 | — |
| `Patrol` | 6 | — |
| `AllPvE` | 7 | — |
| `Reserved9` | 9 | — |
| `Control` | 10 | — |
| `Reserved11` | 11 | — |
| `Clash` | 12 | Clash -> Destiny's name for Team Deathmatch. 4v4 combat, the team with the highest kills at the end of time wins. |
| `Reserved13` | 13 | — |
| `CrimsonDoubles` | 15 | — |
| `Nightfall` | 16 | — |
| `HeroicNightfall` | 17 | — |
| `AllStrikes` | 18 | — |
| `IronBanner` | 19 | — |
| `Reserved20` | 20 | — |
| `Reserved21` | 21 | — |
| `Reserved22` | 22 | — |
| `Reserved24` | 24 | — |
| `AllMayhem` | 25 | — |
| `Reserved26` | 26 | — |
| `Reserved27` | 27 | — |
| `Reserved28` | 28 | — |
| `Reserved29` | 29 | — |
| `Reserved30` | 30 | — |
| `Supremacy` | 31 | — |
| `PrivateMatchesAll` | 32 | — |
| `Survival` | 37 | — |
| `Countdown` | 38 | — |
| `TrialsOfTheNine` | 39 | — |
| `Social` | 40 | — |
| `TrialsCountdown` | 41 | — |
| `TrialsSurvival` | 42 | — |
| `IronBannerControl` | 43 | — |
| `IronBannerClash` | 44 | — |
| `IronBannerSupremacy` | 45 | — |
| `ScoredNightfall` | 46 | — |
| `ScoredHeroicNightfall` | 47 | — |
| `Rumble` | 48 | — |
| `AllDoubles` | 49 | — |
| `Doubles` | 50 | — |
| `PrivateMatchesClash` | 51 | — |
| `PrivateMatchesControl` | 52 | — |
| `PrivateMatchesSupremacy` | 53 | — |
| `PrivateMatchesCountdown` | 54 | — |
| `PrivateMatchesSurvival` | 55 | — |
| `PrivateMatchesMayhem` | 56 | — |
| `PrivateMatchesRumble` | 57 | — |
| `HeroicAdventure` | 58 | — |
| `Showdown` | 59 | — |
| `Lockdown` | 60 | — |
| `Scorched` | 61 | — |
| `ScorchedTeam` | 62 | — |
| `Gambit` | 63 | — |
| `AllPvECompetitive` | 64 | — |
| `Breakthrough` | 65 | — |
| `BlackArmoryRun` | 66 | — |
| `Salvage` | 67 | — |
| `IronBannerSalvage` | 68 | — |
| `PvPCompetitive` | 69 | — |
| `PvPQuickplay` | 70 | — |
| `ClashQuickplay` | 71 | — |
| `ClashCompetitive` | 72 | — |
| `ControlQuickplay` | 73 | — |
| `ControlCompetitive` | 74 | — |
| `GambitPrime` | 75 | — |
| `Reckoning` | 76 | — |
| `Menagerie` | 77 | — |
| `VexOffensive` | 78 | — |
| `NightmareHunt` | 79 | — |
| `Elimination` | 80 | — |
| `Momentum` | 81 | — |
| `Dungeon` | 82 | — |
| `Sundial` | 83 | — |
| `TrialsOfOsiris` | 84 | — |
| `Dares` | 85 | — |
| `Offensive` | 86 | — |
| `LostSector` | 87 | — |
| `Rift` | 88 | — |
| `ZoneControl` | 89 | — |
| `IronBannerRift` | 90 | — |
| `IronBannerZoneControl` | 91 | — |
| `Relic` | 92 | — |

#### Destiny.Definitions.DestinyActivityModeDefinition

**Object** · *(Manifest definition, table `ActivityModes`)*

This definition represents an "Activity Mode" as it exists in the Historical Stats endpoints. An individual Activity Mode represents a collection of activities that are played in a certain way. For example, Nightfall Strikes are part of a "Nightfall" activity mode, and any activities played as the PVP mode "Clash" are part of the "Clash activity mode. Activity modes are nested under each other in a hierarchy, so that if you ask for - for example - "AllPvP", you will get any PVP activities that the user has played, regardless of what specific PVP mode was being played.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `pgcrImage` | string | If this activity mode has a related PGCR image, this will be the path to said image. |
| `modeType` | int32 | The Enumeration value for this Activity Mode. Pass this identifier into Stats endpoints to get aggregate stats for this mode. |
| `activityModeCategory` | int32 | The type of play being performed in broad terms (PVP, PVE) |
| `isTeamBased` | boolean | If True, this mode has oppositional teams fighting against each other rather than "Free-For-All" or Co-operative modes of play. Note that Aggregate modes are never marked as team based, even if they happen to be team based at the moment. At any time, an aggregate whose subordinates are only team based could be changed so that one or more aren't team based, and then this boolean won't make much sense (the aggregation would become "sometimes team based"). Let's not deal with that right now. |
| `isAggregateMode` | boolean | If true, this mode is an aggregation of other, more specific modes rather than being a mode in itself. This includes modes that group Features/Events rather than Gameplay, such as Trials of The Nine: Trials of the Nine being an Event that is interesting to see aggregate data for, but when you play the activities within Trials of the Nine they are more specific activity modes such as Clash. |
| `parentHashes` | array&lt;uint32&gt; | The hash identifiers of the DestinyActivityModeDefinitions that represent all of the "parent" modes for this mode. For instance, the Nightfall Mode is also a member of AllStrikes and AllPvE. |
| `friendlyName` | string | A Friendly identifier you can use for referring to this Activity Mode. We really only used this in our URLs, so... you know, take that for whatever it's worth. |
| `activityModeMappings` | Mapping&lt;uint32, int32&gt; | If this exists, the mode has specific Activities (referred to by the Key) that should instead map to other Activity Modes when they are played. This was useful in D1 for Private Matches, where we wanted to have Private Matches as an activity mode while still referring to the specific mode being played. |
| `display` | boolean | If FALSE, we want to ignore this type when we're showing activity modes in BNet UI. It will still be returned in case 3rd parties want to use it for any purpose. |
| `order` | int32 | The relative ordering of activity modes. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.DestinyActivityModeCategoryEnumeration

**Enum** (`int32`)

Activity Modes are grouped into a few possible broad categories.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | Activities that are neither PVP nor PVE, such as social activities. |
| `PvE` | 1 | PvE activities, where you shoot aliens in the face. |
| `PvP` | 2 | PvP activities, where you shoot your "friends". |
| `PvECompetitive` | 3 | PVE competitive activities, where you shoot whoever you want whenever you want. Or run around collecting small glowing triangles. |

#### Destiny.Definitions.DestinyActivityMatchmakingBlockDefinition

**Type:** object

Information about matchmaking and party size for the activity.

| Property | Type | Description |
| --- | --- | --- |
| `isMatchmade` | boolean | If TRUE, the activity is matchmade. Otherwise, it requires explicit forming of a party. |
| `minParty` | int32 | The minimum # of people in the fireteam for the activity to launch. |
| `maxParty` | int32 | The maximum # of people allowed in a Fireteam. |
| `maxPlayers` | int32 | The maximum # of people allowed across all teams in the activity. |
| `requiresGuardianOath` | boolean | If true, you have to Solemnly Swear to be up to Nothing But Good(tm) to play. |

#### Destiny.Definitions.DestinyActivityGuidedBlockDefinition

**Type:** object

Guided Game information for this activity.

| Property | Type | Description |
| --- | --- | --- |
| `guidedMaxLobbySize` | int32 | The maximum amount of people that can be in the waiting lobby. |
| `guidedMinLobbySize` | int32 | The minimum amount of people that can be in the waiting lobby. |
| `guidedDisbandCount` | int32 | If -1, the guided group cannot be disbanded. Otherwise, take the total # of players in the activity and subtract this number: that is the total # of votes needed for the guided group to disband. |

#### Destiny.Definitions.DestinyActivityLoadoutRequirementSet

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `requirements` | array&lt;Destiny.Definitions.DestinyActivityLoadoutRequirement&gt; | The set of requirements that will be applied on the activity if this requirement set is active. |

#### Destiny.Definitions.DestinyActivityLoadoutRequirement

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `equipmentSlotHash` | uint32 → DestinyEquipmentSlotDefinition | — |
| `allowedEquippedItemHashes` | array&lt;uint32&gt; → DestinyInventoryItemDefinition | — |
| `allowedWeaponSubTypes` | array&lt;int32&gt; | — |

#### Destiny.DestinyItemSubTypeEnumeration

**Enum** (`int32`)

This Enumeration further classifies items by more specific categorizations than DestinyItemType. The "Sub-Type" is where we classify and categorize items one step further in specificity: "Auto Rifle" instead of just "Weapon" for example, or "Vanguard Bounty" instead of merely "Bounty". These sub-types are provided for historical compatibility with Destiny 1, but an ideal alternative is to use DestinyItemCategoryDefinitions and the DestinyItemDefinition.itemCategories property instead. Item Categories allow for arbitrary hierarchies of specificity, and for items to belong to multiple categories across multiple hierarchies simultaneously. For this enum, we pick a single type as a "best guess" fit. NOTE: This is not all of the item types available, and some of these are holdovers from Destiny 1 that may or may not still exist.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Crucible` | 1 | DEPRECATED. Items can be both "Crucible" and something else interesting. |
| `Vanguard` | 2 | DEPRECATED. An item can both be "Vanguard" and something else. |
| `Exotic` | 5 | DEPRECATED. An item can both be Exotic and something else. |
| `AutoRifle` | 6 | — |
| `Shotgun` | 7 | — |
| `Machinegun` | 8 | — |
| `HandCannon` | 9 | — |
| `RocketLauncher` | 10 | — |
| `FusionRifle` | 11 | — |
| `SniperRifle` | 12 | — |
| `PulseRifle` | 13 | — |
| `ScoutRifle` | 14 | — |
| `Crm` | 16 | DEPRECATED. An item can both be CRM and something else. |
| `Sidearm` | 17 | — |
| `Sword` | 18 | — |
| `Mask` | 19 | — |
| `Shader` | 20 | — |
| `Ornament` | 21 | — |
| `FusionRifleLine` | 22 | — |
| `GrenadeLauncher` | 23 | — |
| `SubmachineGun` | 24 | — |
| `TraceRifle` | 25 | — |
| `HelmetArmor` | 26 | — |
| `GauntletsArmor` | 27 | — |
| `ChestArmor` | 28 | — |
| `LegArmor` | 29 | — |
| `ClassArmor` | 30 | — |
| `Bow` | 31 | — |
| `DummyRepeatableBounty` | 32 | — |
| `Glaive` | 33 | — |

#### Destiny.Definitions.DestinyActivityInsertionPointDefinition

**Type:** object

A point of entry into an activity, gated by an unlock flag and with some more-or-less useless (for our purposes) phase information. I'm including it in case we end up being able to bolt more useful information onto it in the future. UPDATE: Turns out this information isn't actually useless, and is in fact actually useful for people. Who would have thought? We still don't have localized info for it, but at least this will help people when they're looking at phase indexes in stats data, or when they want to know what phases have been completed on a weekly achievement.

| Property | Type | Description |
| --- | --- | --- |
| `phaseHash` | uint32 | A unique hash value representing the phase. This can be useful for, for example, comparing how different instances of Raids have phases in different orders! |

#### Destiny.Constants.DestinyEnvironmentLocationMapping

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `locationHash` | uint32 → DestinyLocationDefinition | The location that is revealed on the director by this mapping. |
| `activationSource` | string | A hint that the UI uses to figure out how this location is activated by the player. |
| `itemHash` | uint32 → DestinyInventoryItemDefinition? | If this is populated, it is the item that you must possess for this location to be active because of this mapping. (theoretically, a location can have multiple mappings, and some might require an item while others don't) |
| `objectiveHash` | uint32 → DestinyObjectiveDefinition? | If this is populated, this is an objective related to the location. |
| `activityHash` | uint32 → DestinyActivityDefinition? | If this is populated, this is the activity you have to be playing in order to see this location appear because of this mapping. (theoretically, a location can have multiple mappings, and some might require you to be in a specific activity when others don't) |

#### Destiny.Definitions.DestinyActivityCuratorBlockDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `quickplaySortPriority` | int32 | Sort order |
| `quickplaySortToFront` | boolean | Whether this activity should be sorted to the front of the Portal category |

#### Destiny.Definitions.DestinyActivityDurationEstimate

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `durationPipsFilledCount` | int32 | The number of filled pips shown on the Portal tile |
| `durationPipsTotalCount` | int32 | The total number of pips shown on the Portal tile |
| `durationEstimateText` | string | The text string showing the estimated time to complete this activity |

#### Destiny.Definitions.DestinyPlaceDefinition

**Object** · *(Manifest definition, table `Places`)*

Okay, so Activities (DestinyActivityDefinition) take place in Destinations (DestinyDestinationDefinition). Destinations are part of larger locations known as Places (you're reading its documentation right now). Places are more on the planetary scale, like "Earth" and "Your Mom."

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyActivityTypeDefinition

**Object** · *(Manifest definition, table `ActivityTypes`)*

The definition for an Activity Type. In Destiny 2, an Activity Type represents a conceptual categorization of Activities. These are most commonly used in the game for the subtitle under Activities, but BNet uses them extensively to identify and group activities by their common properties. Unfortunately, there has been a movement away from providing the richer data in Destiny 2 that we used to get in Destiny 1 for Activity Types. For instance, Nightfalls are grouped under the same Activity Type as regular Strikes. For this reason, BNet will eventually migrate toward Activity Modes as a better indicator of activity category. But for the time being, it is still referred to in many places across our codebase.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Activities.DestinyActivityFamilyDefinition

**Object** · *(Manifest definition, table `ActivityFamilies`)*

| Property | Type | Description |
| --- | --- | --- |
| `traits` | array&lt;uint32&gt; → DestinyTraitDefinition | — |
| `disabledSkullCategoryHashes` | array&lt;uint32&gt; → DestinyActivitySkullCategoryDefinition | — |
| `disabledSkullSubcategoryHashes` | array&lt;uint32&gt; → DestinyActivitySkullSubcategoryDefinition | — |
| `fixedSkullSubcategoryHashes` | array&lt;uint32&gt; → DestinyActivitySkullSubcategoryDefinition | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Traits.DestinyTraitDefinition

**Object** · *(Manifest definition, table `Traits`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `displayHint` | string | An identifier for how this trait can be displayed. For example: a 'keyword' hint to show an explanation for certain related terms. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Activities.DestinyActivitySkullCategoryDefinition

**Object** · *(Manifest definition, table `ActivitySkullCategories`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Activities.DestinyActivitySkullSubcategoryDefinition

**Object** · *(Manifest definition, table `ActivitySkullSubcategories`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `parentSkullCategoryHash` | uint32 → DestinyActivitySkullCategoryDefinition | — |
| `availabilityTierRank` | int32 | — |
| `defaultSkullHashes` | array&lt;uint32&gt; | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Activities.DestinyActivityDifficultyTierCollectionDefinition

**Object** · *(Manifest definition, table `ActivityDifficultyTierCollections`)*

| Property | Type | Description |
| --- | --- | --- |
| `difficultyTiers` | array&lt;Destiny.Definitions.Activities.DestinyActivityDifficultyTierDefinition&gt; | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Activities.DestinyActivityDifficultyTierDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `recommendedActivityLevelOffset` | int32 | — |
| `fixedActivitySkulls` | array&lt;Destiny.Definitions.Activities.DestinyActivitySkull&gt; | — |
| `tierType` | int32 | — |
| `optionalRequiredTrait` | uint32 → DestinyTraitDefinition? | — |
| `activityLevel` | int32 | — |
| `tierRank` | int32 | — |
| `minimumFireteamLeaderPower` | int32 | — |
| `maximumFireteamLeaderPower` | int32 | — |
| `scoreTimeLimitMultiplier` | int32 | — |
| `selectableSkullCollectionHashes` | array&lt;uint32&gt; → DestinyActivitySelectableSkullCollectionDefinition | — |
| `skullSubcategoryOverrides` | array&lt;Destiny.Definitions.Activities.DestinyActivityDifficultyTierSubcategoryOverride&gt; | — |

#### Destiny.Definitions.Activities.DestinyActivitySkull

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `hash` | uint32 | — |
| `skullIdentifierHash` | uint32 | — |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `skullOptions` | array&lt;Destiny.Definitions.Activities.DestinyActivitySkullOption&gt; | — |
| `dynamicUse` | int32 | — |
| `modifierPowerContribution` | int32 | — |
| `modifierMultiplierContribution` | float | — |
| `skullExclusionGroupHash` | uint32 → DestinyActivitySelectableSkullExclusionGroupDefinition? | — |
| `hasUi` | boolean | — |
| `displayDescriptionOverrideForNavMode` | string | — |
| `activityModifierDisplayCategory` | int32 | — |
| `activityModifierConnotation` | int32 | — |
| `displayInNavMode` | boolean | — |
| `displayInActivitySelection` | boolean | — |

#### Destiny.Definitions.Activities.DestinyActivitySkullOption

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `optionHash` | uint32 | — |
| `stringValue` | string | — |
| `boolValue` | boolean | — |
| `integerValue` | int32 | — |
| `floatValue` | float | — |
| `minDisplayDifficultyId` | int32 | — |

#### Destiny.DestinyActivityDifficultyIdEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Trivial` | 0 | — |
| `Easy` | 1 | — |
| `Normal` | 2 | — |
| `Challenging` | 3 | — |
| `Hard` | 4 | — |
| `Brave` | 5 | — |
| `AlmostImpossible` | 6 | — |
| `Impossible` | 7 | — |
| `Count` | 8 | — |

#### Destiny.DestinyActivitySkullDynamicUseEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Unknown` | 0 | — |
| `Allowed` | 1 | — |
| `Disallowed` | 2 | — |
| `Count` | 3 | — |

#### Destiny.DestinyActivityModifierDisplayCategoryEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `ModeRules` | 1 | — |
| `SelfBuildcraft` | 2 | — |
| `EnemyAdjustment` | 3 | — |
| `EnemyBuildcraft` | 4 | — |
| `Seasonal` | 5 | — |
| `Fun` | 6 | — |
| `Count` | 7 | — |

#### Destiny.DestinyActivityModifierConnotationEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Neutral` | 0 | — |
| `Positive` | 1 | — |
| `Negative` | 2 | — |
| `Affix` | 3 | — |
| `Informational` | 4 | — |
| `Reward` | 5 | — |
| `Event` | 6 | — |
| `Count` | 7 | — |

#### Destiny.Definitions.Activities.DestinyActivitySelectableSkullExclusionGroupDefinition

**Object** · *(Manifest definition, table `ActivitySelectableSkullExclusionGroups`)*

| Property | Type | Description |
| --- | --- | --- |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.DestinyActivityDifficultyTierTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Default` | 0 | — |
| `Training` | 1 | — |
| `Count` | 2 | — |

#### Destiny.Definitions.Activities.DestinyActivityDifficultyTierSubcategoryOverride

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `skullSubcategoryHash` | uint32 | — |
| `refreshTimeMinutes` | int32 | — |
| `refreshTimeOffsetMinutes` | int32 | — |

#### Destiny.Definitions.Activities.DestinyActivitySelectableSkullCollectionDefinition

**Object** · *(Manifest definition, table `ActivitySelectableSkullCollections`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `skullSubcategoryHashes` | array&lt;uint32&gt; → DestinyActivitySkullSubcategoryDefinition | — |
| `selectionType` | Destiny.Definitions.Activities.DestinyActivitySelectableSkullCollectionSelectionType | — |
| `selectableActivitySkulls` | array&lt;Destiny.Definitions.Activities.DestinyActivitySelectableSkull&gt; | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Activities.DestinyActivitySelectableSkullCollectionSelectionType

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `selectionCount` | int32 | — |
| `refreshTimeMinutes` | int32 | — |
| `refreshTimeOffsetMinutes` | int32 | — |

#### Destiny.Definitions.Activities.DestinyActivitySelectableSkull

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `requiredTraitHash` | uint32 → DestinyTraitDefinition? | — |
| `requiredTraitExistence` | boolean | — |
| `isEmptySkull` | boolean | — |
| `loadoutRestrictionHash` | uint32 → DestinyActivityLoadoutRestrictionDefinition? | — |
| `activitySkull` | Destiny.Definitions.Activities.DestinyActivitySkull | — |

#### Destiny.Definitions.Activities.DestinyActivityLoadoutRestrictionDefinition

**Object** · *(Manifest definition, table `ActivityLoadoutRestrictionDefinitions`)*

| Property | Type | Description |
| --- | --- | --- |
| `restrictedItemFilterHash` | uint32 | — |
| `restrictedEquipmentSlotHashes` | array&lt;uint32&gt; → DestinyEquipmentSlotDefinition | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Director.DestinyActivityGraphNodeStateEntry

**Type:** object

Represents a single state that a graph node might end up in. Depending on what's going on in the game, graph nodes could be shown in different ways or even excluded from view entirely.

| Property | Type | Description |
| --- | --- | --- |
| `state` | int32 | — |

#### Destiny.DestinyGraphNodeStateEnumeration

**Enum** (`int32`)

Represents a potential state of an Activity Graph node.

| Value | # | Description |
| --- | --- | --- |
| `Hidden` | 0 | — |
| `Visible` | 1 | — |
| `Teaser` | 2 | — |
| `Incomplete` | 3 | — |
| `Completed` | 4 | — |

#### Destiny.Definitions.Director.DestinyActivityGraphArtElementDefinition

**Type:** object

These Art Elements are meant to represent one-off visual effects overlaid on the map. Currently, we do not have a pipeline to import the assets for these overlays, so this info exists as a placeholder for when such a pipeline exists (if it ever will)

| Property | Type | Description |
| --- | --- | --- |
| `position` | Destiny.Definitions.Common.DestinyPositionDefinition | The position on the map of the art element. |

#### Destiny.Definitions.Director.DestinyActivityGraphConnectionDefinition

**Type:** object

Nodes on a graph can be visually connected: this appears to be the information about which nodes to link. It appears to lack more detailed information, such as the path for that linking.

| Property | Type | Description |
| --- | --- | --- |
| `sourceNodeHash` | uint32 | — |
| `destNodeHash` | uint32 | — |

#### Destiny.Definitions.Director.DestinyActivityGraphDisplayObjectiveDefinition

**Type:** object

When a Graph needs to show active Objectives, this defines those objectives as well as an identifier.

| Property | Type | Description |
| --- | --- | --- |
| `id` | uint32 | $NOTE $amola 2017-01-19 This field is apparently something that CUI uses to manually wire up objectives to display info. I am unsure how it works. |
| `objectiveHash` | uint32 → DestinyObjectiveDefinition | The objective being shown on the map. |

#### Destiny.Definitions.Director.DestinyActivityGraphDisplayProgressionDefinition

**Type:** object

When a Graph needs to show active Progressions, this defines those objectives as well as an identifier.

| Property | Type | Description |
| --- | --- | --- |
| `id` | uint32 | — |
| `progressionHash` | uint32 | — |

#### Destiny.Definitions.Director.DestinyLinkedGraphDefinition

**Type:** object

This describes links between the current graph and others, as well as when that link is relevant.

| Property | Type | Description |
| --- | --- | --- |
| `description` | string | — |
| `name` | string | — |
| `linkedGraphId` | uint32 | — |
| `linkedGraphs` | array&lt;Destiny.Definitions.Director.DestinyLinkedGraphEntryDefinition&gt; | — |
| `overview` | string | — |

#### Destiny.Definitions.Director.DestinyLinkedGraphEntryDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `activityGraphHash` | uint32 | — |

#### Destiny.Definitions.DestinyDestinationBubbleSettingDefinition

**Type:** object

Human readable data about the bubble. Combine with DestinyBubbleDefinition - see DestinyDestinationDefinition.bubbleSettings for more information. DEPRECATED - Just use bubbles.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |

#### Destiny.Definitions.DestinyBubbleDefinition

**Type:** object

Basic identifying data about the bubble. Combine with DestinyDestinationBubbleSettingDefinition - see DestinyDestinationDefinition.bubbleSettings for more information.

| Property | Type | Description |
| --- | --- | --- |
| `hash` | uint32 | The identifier for the bubble: only guaranteed to be unique within the Destination. |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | The display properties of this bubble, so you don't have to look them up in a separate list anymore. |

#### Destiny.Definitions.DestinyVendorGroupReference

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `vendorGroupHash` | uint32 → DestinyVendorGroupDefinition | The DestinyVendorGroupDefinition to which this Vendor can belong. |

#### Destiny.Definitions.DestinyVendorGroupDefinition

**Object** · *(Manifest definition, table `VendorGroups`)*

BNet attempts to group vendors into similar collections. These groups aren't technically game canonical, but they are helpful for filtering vendors or showing them organized into a clean view on a webpage or app. These definitions represent the groups we've built. Unlike in Destiny 1, a Vendors' group may change dynamically as the game state changes: thus, you will want to check DestinyVendorComponent responses to find a vendor's currently active Group (if you care). Using this will let you group your vendors in your UI in a similar manner to how we will do grouping in the Companion.

| Property | Type | Description |
| --- | --- | --- |
| `order` | int32 | The recommended order in which to render the groups, Ascending order. |
| `categoryName` | string | For now, a group just has a name. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyFactionDefinition

**Object** · *(Manifest definition, table `Factions`)*

These definitions represent Factions in the game. Factions have ended up unilaterally being related to Vendors that represent them, but that need not necessarily be the case. A Faction is really just an entity that has a related progression for which a character can gain experience. In Destiny 1, Dead Orbit was an example of a Faction: there happens to be a Vendor that represents Dead Orbit (and indeed, DestinyVendorDefinition.factionHash defines to this relationship), but Dead Orbit could theoretically exist without the Vendor that provides rewards.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `progressionHash` | uint32 → DestinyProgressionDefinition | The hash identifier for the DestinyProgressionDefinition that indicates the character's relationship with this faction in terms of experience and levels. |
| `tokenValues` | Mapping&lt;uint32, uint32&gt; | The faction token item hashes, and their respective progression values. |
| `rewardItemHash` | uint32 → DestinyInventoryItemDefinition | The faction reward item hash, usually an engram. |
| `rewardVendorHash` | uint32 → DestinyVendorDefinition | The faction reward vendor hash, used for faction engram previews. |
| `vendors` | array&lt;Destiny.Definitions.DestinyFactionVendorDefinition&gt; | List of vendors that are associated with this faction. The last vendor that passes the unlock flag checks is the one that should be shown. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyFactionVendorDefinition

**Type:** object

These definitions represent faction vendors at different points in the game. A single faction may contain multiple vendors, or the same vendor available at two different locations.

| Property | Type | Description |
| --- | --- | --- |
| `vendorHash` | uint32 → DestinyVendorDefinition | The faction vendor hash. |
| `destinationHash` | uint32 → DestinyDestinationDefinition | The hash identifier for a Destination at which this vendor may be located. Each destination where a Vendor may exist will only ever have a single entry. |
| `backgroundImagePath` | string | The relative path to the background image representing this Vendor at this location, for use in a banner. |

#### Destiny.Definitions.DestinySandboxPatternDefinition

**Object** · *(Manifest definition, table `SandboxPatterns`)*

| Property | Type | Description |
| --- | --- | --- |
| `patternHash` | uint32 | — |
| `patternGlobalTagIdHash` | uint32 | — |
| `weaponContentGroupHash` | uint32 | — |
| `weaponTranslationGroupHash` | uint32 | — |
| `weaponTypeHash` | uint32? | — |
| `weaponType` | int32 | — |
| `filters` | array&lt;Destiny.Definitions.DestinyArrangementRegionFilterDefinition&gt; | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyArrangementRegionFilterDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `artArrangementRegionHash` | uint32 | — |
| `artArrangementRegionIndex` | int32 | — |
| `statHash` | uint32 | — |
| `arrangementIndexByStatValue` | Mapping&lt;int32, int32&gt; | — |

#### Destiny.Definitions.DestinyItemPreviewBlockDefinition

**Type:** object

Items like Sacks or Boxes can have items that it shows in-game when you view details that represent the items you can obtain if you use or acquire the item. This defines those categories, and gives some insights into that data's source.

| Property | Type | Description |
| --- | --- | --- |
| `screenStyle` | string | A string that the game UI uses as a hint for which detail screen to show for the item. You, too, can leverage this for your own custom screen detail views. Note, however, that these are arbitrarily defined by designers: there's no guarantees of a fixed, known number of these - so fall back to something reasonable if you don't recognize it. |
| `previewVendorHash` | uint32 → DestinyVendorDefinition | If the preview data is derived from a fake "Preview" Vendor, this will be the hash identifier for the DestinyVendorDefinition of that fake vendor. |
| `artifactHash` | uint32 → DestinyArtifactDefinition? | If this item should show you Artifact information when you preview it, this is the hash identifier of the DestinyArtifactDefinition for the artifact whose data should be shown. |
| `previewActionString` | string | If the preview has an associated action (like "Open"), this will be the localized string for that action. |
| `derivedItemCategories` | array&lt;Destiny.Definitions.Items.DestinyDerivedItemCategoryDefinition&gt; | This is a list of the items being previewed, categorized in the same way as they are in the preview UI. |

#### Destiny.Definitions.Items.DestinyDerivedItemCategoryDefinition

**Type:** object

A shortcut for the fact that some items have a "Preview Vendor" - See DestinyInventoryItemDefinition.preview.previewVendorHash - that is intended to be used to show what items you can get as a result of acquiring or using this item. A common example of this in Destiny 1 was Eververse "Boxes," which could have many possible items. This "Preview Vendor" is not a vendor you can actually see in the game, but it defines categories and sale items for all of the possible items you could get from the Box so that the game can show them to you. We summarize that info here so that you don't have to do that Vendor lookup and aggregation manually.

| Property | Type | Description |
| --- | --- | --- |
| `categoryDescription` | string | The localized string for the category title. This will be something describing the items you can get as a group, or your likelihood/the quantity you'll get. |
| `items` | array&lt;Destiny.Definitions.Items.DestinyDerivedItemDefinition&gt; | This is the list of all of the items for this category and the basic properties we'll know about them. |

#### Destiny.Definitions.Items.DestinyDerivedItemDefinition

**Type:** object

This is a reference to, and summary data for, a specific item that you can get as a result of Using or Acquiring some other Item (For example, this could be summary information for an Emote that you can get by opening an an Eververse Box) See DestinyDerivedItemCategoryDefinition for more information.

| Property | Type | Description |
| --- | --- | --- |
| `itemHash` | uint32? | The hash for the DestinyInventoryItemDefinition of this derived item, if there is one. Sometimes we are given this information as a manual override, in which case there won't be an actual DestinyInventoryItemDefinition for what we display, but you can still show the strings from this object itself. |
| `itemName` | string | The name of the derived item. |
| `itemDetail` | string | Additional details about the derived item, in addition to the description. |
| `itemDescription` | string | A brief description of the item. |
| `iconPath` | string | An icon for the item. |
| `vendorItemIndex` | int32 | If the item was derived from a "Preview Vendor", this will be an index into the DestinyVendorDefinition's itemList property. Otherwise, -1. |

#### Destiny.Definitions.Artifacts.DestinyArtifactDefinition

**Object** · *(Manifest definition, table `Artifacts`)*

Represents known info about a Destiny Artifact. We cannot guarantee that artifact definitions will be immutable between seasons - in fact, we've been told that they will be replaced between seasons. But this definition is built both to minimize the amount of lookups for related data that have to occur, and is built in hope that, if this plan changes, we will be able to accommodate it more easily.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | Any basic display info we know about the Artifact. Currently sourced from a related inventory item, but the source of this data is subject to change. |
| `translationBlock` | Destiny.Definitions.DestinyItemTranslationBlockDefinition | Any Geometry/3D info we know about the Artifact. Currently sourced from a related inventory item's gearset information, but the source of this data is subject to change. |
| `tiers` | array&lt;Destiny.Definitions.Artifacts.DestinyArtifactTierDefinition&gt; | Any Tier/Rank data related to this artifact, listed in display order. Currently sourced from a Vendor, but this source is subject to change. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Artifacts.DestinyArtifactTierDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `tierHash` | uint32 | An identifier, unique within the Artifact, for this specific tier. |
| `displayTitle` | string | The human readable title of this tier, if any. |
| `progressRequirementMessage` | string | A string representing the localized minimum requirement text for this Tier, if any. |
| `items` | array&lt;Destiny.Definitions.Artifacts.DestinyArtifactTierItemDefinition&gt; | The items that can be earned within this tier. |
| `minimumUnlockPointsUsedRequirement` | int32 | The minimum number of "unlock points" that you must have used before you can unlock items from this tier. |

#### Destiny.Definitions.Artifacts.DestinyArtifactTierItemDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | The identifier of the Plug Item unlocked by activating this item in the Artifact. |

#### Destiny.Definitions.DestinyItemQualityBlockDefinition

**Type:** object

An item's "Quality" determines its calculated stats. The Level at which the item spawns is combined with its "qualityLevel" along with some additional calculations to determine the value of those stats. In Destiny 2, most items don't have default item levels and quality, making this property less useful: these apparently are almost always determined by the complex mechanisms of the Reward system rather than statically. They are still provided here in case they are still useful for people. This also contains some information about Infusion.

| Property | Type | Description |
| --- | --- | --- |
| `itemLevels` | array&lt;int32&gt; | The "base" defined level of an item. This is a list because, in theory, each Expansion could define its own base level for an item. In practice, not only was that never done in Destiny 1, but now this isn't even populated at all. When it's not populated, the level at which it spawns has to be inferred by Reward information, of which BNet receives an imperfect view and will only be reliable on instanced data as a result. |
| `qualityLevel` | int32 | qualityLevel is used in combination with the item's level to calculate stats like Attack and Defense. It plays a role in that calculation, but not nearly as large as itemLevel does. |
| `infusionCategoryName` | string | The string identifier for this item's "infusability", if any. Items that match the same infusionCategoryName are allowed to infuse with each other. DEPRECATED: Items can now have multiple infusion categories. Please use infusionCategoryHashes instead. |
| `infusionCategoryHash` | uint32 | The hash identifier for the infusion. It does not map to a Definition entity. DEPRECATED: Items can now have multiple infusion categories. Please use infusionCategoryHashes instead. |
| `infusionCategoryHashes` | array&lt;uint32&gt; | If any one of these hashes matches any value in another item's infusionCategoryHashes, the two can infuse with each other. |
| `progressionLevelRequirementHash` | uint32 → DestinyProgressionLevelRequirementDefinition | An item can refer to pre-set level requirements. They are defined in DestinyProgressionLevelRequirementDefinition, and you can use this hash to find the appropriate definition. |
| `currentVersion` | uint32 | The latest version available for this item. |
| `versions` | array&lt;Destiny.Definitions.DestinyItemVersionDefinition&gt; | The list of versions available for this item. |
| `displayVersionWatermarkIcons` | array&lt;string&gt; | Icon overlays to denote the item version and power cap status. |

#### Destiny.Definitions.DestinyItemVersionDefinition

**Type:** object

The version definition currently just holds a reference to the power cap.

| Property | Type | Description |
| --- | --- | --- |
| `powerCapHash` | uint32 → DestinyPowerCapDefinition | A reference to the power cap for this item version. |

#### Destiny.Definitions.PowerCaps.DestinyPowerCapDefinition

**Object** · *(Manifest definition, table `PowerCaps`)*

Defines a 'power cap' (limit) for gear items, based on the rarity tier and season of release.

| Property | Type | Description |
| --- | --- | --- |
| `powerCap` | int32 | The raw value for a power cap. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Progression.DestinyProgressionLevelRequirementDefinition

**Object** · *(Manifest definition, table `ProgressionLevelRequirements`)*

These are pre-constructed collections of data that can be used to determine the Level Requirement for an item given a Progression to be tested (such as the Character's level). For instance, say a character receives a new Auto Rifle, and that Auto Rifle's DestinyInventoryItemDefinition.quality.progressionLevelRequirementHash property is pointing at one of these DestinyProgressionLevelRequirementDefinitions. Let's pretend also that the progressionHash it is pointing at is the Character Level progression. In that situation, the character's level will be used to interpolate a value in the requirementCurve property. The value picked up from that interpolation will be the required level for the item.

| Property | Type | Description |
| --- | --- | --- |
| `requirementCurve` | array&lt;Interpolation.InterpolationPointFloat&gt; | A curve of level requirements, weighted by the related progressions' level. Interpolate against this curve with the character's progression level to determine what the level requirement of the generated item that is using this data will be. |
| `progressionHash` | uint32 → DestinyProgressionDefinition | The progression whose level should be used to determine the level requirement. Look up the DestinyProgressionDefinition with this hash for more information about the progression in question. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyItemValueBlockDefinition

**Type:** object

This defines an item's "Value". Unfortunately, this appears to be used in different ways depending on the way that the item itself is used. For items being sold at a Vendor, this is the default "sale price" of the item. These days, the vendor itself almost always sets the price, but it still possible for the price to fall back to this value. For quests, it is a preview of rewards you can gain by completing the quest. For dummy items, if the itemValue refers to an Emblem, it is the emblem that should be shown as the reward. (jeez louise) It will likely be used in a number of other ways in the future, it appears to be a bucket where they put arbitrary items and quantities into the item.

| Property | Type | Description |
| --- | --- | --- |
| `itemValue` | array&lt;Destiny.DestinyItemQuantity&gt; | References to the items that make up this item's "value", and the quantity. |
| `valueDescription` | string | If there's a localized text description of the value provided, this will be said description. |

#### Destiny.Definitions.DestinyItemSourceBlockDefinition

**Type:** object

Data about an item's "sources": ways that the item can be obtained.

| Property | Type | Description |
| --- | --- | --- |
| `sourceHashes` | array&lt;uint32&gt; → DestinyRewardSourceDefinition | The list of hash identifiers for Reward Sources that hint where the item can be found (DestinyRewardSourceDefinition). |
| `sources` | array&lt;Destiny.Definitions.Sources.DestinyItemSourceDefinition&gt; | A collection of details about the stats that were computed for the ways we found that the item could be spawned. |
| `exclusive` | int32 | If we found that this item is exclusive to a specific platform, this will be set to the BungieMembershipType enumeration that matches that platform. |
| `vendorSources` | array&lt;Destiny.Definitions.DestinyItemVendorSourceReference&gt; | A denormalized reference back to vendors that potentially sell this item. |

#### Destiny.Definitions.Sources.DestinyItemSourceDefinition

**Type:** object

Properties of a DestinyInventoryItemDefinition that store all of the information we were able to discern about how the item spawns, and where you can find the item. Items will have many of these sources, one per level at which it spawns, to try and give more granular data about where items spawn for specific level ranges.

| Property | Type | Description |
| --- | --- | --- |
| `level` | int32 | The level at which the item spawns. Essentially the Primary Key for this source data: there will be multiple of these source entries per item that has source data, grouped by the level at which the item spawns. |
| `minQuality` | int32 | The minimum Quality at which the item spawns for this level. Examine DestinyInventoryItemDefinition for more information about what Quality means. Just don't ask Phaedrus about it, he'll never stop talking and you'll have to write a book about it. |
| `maxQuality` | int32 | The maximum quality at which the item spawns for this level. |
| `minLevelRequired` | int32 | The minimum Character Level required for equipping the item when the item spawns at the item level defined on this DestinyItemSourceDefinition, as far as we saw in our processing. |
| `maxLevelRequired` | int32 | The maximum Character Level required for equipping the item when the item spawns at the item level defined on this DestinyItemSourceDefinition, as far as we saw in our processing. |
| `computedStats` | Mapping&lt;uint32, Destiny.Definitions.DestinyInventoryItemStatDefinition&gt; | The stats computed for this level/quality range. |
| `sourceHashes` | array&lt;uint32&gt; → DestinyRewardSourceDefinition | The DestinyRewardSourceDefinitions found that can spawn the item at this level. |

#### Destiny.Definitions.DestinyRewardSourceDefinition

**Object** · *(Manifest definition, table `RewardSources`)*

Represents a heuristically-determined "item source" according to Bungie.net. These item sources are non-canonical: we apply a combination of special configuration and often-fragile heuristics to attempt to discern whether an item should be part of a given "source," but we have known cases of false positives and negatives due to our imperfect heuristics. Still, they provide a decent approximation for people trying to figure out how an item can be obtained. DestinyInventoryItemDefinition refers to sources in the sourceDatas.sourceHashes property for all sources we determined the item could spawn from. An example in Destiny 1 of a Source would be "Nightfall". If an item has the "Nightfall" source associated with it, it's extremely likely that you can earn that item while playing Nightfall, either during play or as an after-completion reward.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `category` | int32 | Sources are grouped into categories: common ways that items are provided. I hope to see this expand in Destiny 2 once we have time to generate accurate reward source data. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyRewardSourceCategoryEnumeration

**Enum** (`int32`)

BNet's custom categorization of reward sources. We took a look at the existing ways that items could be spawned, and tried to make high-level categorizations of them. This needs to be re-evaluated for Destiny 2.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | The source doesn't fit well into any of the other types. |
| `Activity` | 1 | The source is directly related to the rewards gained by playing an activity or set of activities. This currently includes Quests and other action in-game. |
| `Vendor` | 2 | This source is directly related to items that Vendors sell. |
| `Aggregate` | 3 | This source is a custom aggregation of items that can be earned in many ways, but that share some other property in common that is useful to share. For instance, in Destiny 1 we would make "Reward Sources" for every game expansion: that way, you could search reward sources to see what items became available with any given Expansion. |

#### Destiny.Definitions.DestinyItemVendorSourceReference

**Type:** object

Represents that a vendor could sell this item, and provides a quick link to that vendor and sale item. Note that we do not and cannot make a guarantee that the vendor will ever *actually* sell this item, only that the Vendor has a definition that indicates it *could* be sold. Note also that a vendor may sell the same item in multiple "ways", which means there may be multiple vendorItemIndexes for a single Vendor hash.

| Property | Type | Description |
| --- | --- | --- |
| `vendorHash` | uint32 → DestinyVendorDefinition | The identifier for the vendor that may sell this item. |
| `vendorItemIndexes` | array&lt;int32&gt; | The Vendor sale item indexes that represent the sale information for this item. The same vendor may sell an item in multiple "ways", hence why this is a list. (for instance, a weapon may be "sold" as a reward in a quest, for Glimmer, and for Masterwork Cores: each of those ways would be represented by a different vendor sale item with a different index) |

#### Destiny.Definitions.DestinyItemObjectiveBlockDefinition

**Type:** object

An item can have objectives on it. In practice, these are the exclusive purview of "Quest Step" items: DestinyInventoryItemDefinitions that represent a specific step in a Quest. Quest steps have 1:M objectives that we end up processing and returning in live data as DestinyQuestStatus data, and other useful information.

| Property | Type | Description |
| --- | --- | --- |
| `objectiveHashes` | array&lt;uint32&gt; → DestinyObjectiveDefinition | The hashes to Objectives (DestinyObjectiveDefinition) that are part of this Quest Step, in the order that they should be rendered. |
| `displayActivityHashes` | array&lt;uint32&gt; → DestinyActivityDefinition | For every entry in objectiveHashes, there is a corresponding entry in this array at the same index. If the objective is meant to be associated with a specific DestinyActivityDefinition, there will be a valid hash at that index. Otherwise, it will be invalid (0). Rendered somewhat obsolete by perObjectiveDisplayProperties, which currently has much the same information but may end up with more info in the future. |
| `requireFullObjectiveCompletion` | boolean | If True, all objectives must be completed for the step to be completed. If False, any one objective can be completed for the step to be completed. |
| `questlineItemHash` | uint32 → DestinyInventoryItemDefinition | The hash for the DestinyInventoryItemDefinition representing the Quest to which this Quest Step belongs. |
| `narrative` | string | The localized string for narrative text related to this quest step, if any. |
| `objectiveVerbName` | string | The localized string describing an action to be performed associated with the objectives, if any. |
| `questTypeIdentifier` | string | The identifier for the type of quest being performed, if any. Not associated with any fixed definition, yet. |
| `questTypeHash` | uint32 | A hashed value for the questTypeIdentifier, because apparently I like to be redundant. |
| `perObjectiveDisplayProperties` | array&lt;Destiny.Definitions.DestinyObjectiveDisplayProperties&gt; | One entry per Objective on the item, it will have related display information. |
| `displayAsStatTracker` | boolean | — |

#### Destiny.Definitions.DestinyObjectiveDisplayProperties

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `activityHash` | uint32 → DestinyActivityDefinition? | The activity associated with this objective in the context of this item, if any. |
| `displayOnItemPreviewScreen` | boolean | If true, the game shows this objective on item preview screens. |

#### Destiny.Definitions.DestinyItemMetricBlockDefinition

**Type:** object

The metrics available for display and selection on an item.

| Property | Type | Description |
| --- | --- | --- |
| `availableMetricCategoryNodeHashes` | array&lt;uint32&gt; → DestinyPresentationNodeDefinition | Hash identifiers for any DestinyPresentationNodeDefinition entry that can be used to list available metrics. Any metric listed directly below these nodes, or in any of these nodes' children will be made available for selection. |

#### Destiny.Definitions.Presentation.DestinyPresentationNodeBaseDefinition

**Type:** object

This is the base class for all presentation system children. Presentation Nodes, Records, Collectibles, and Metrics.

| Property | Type | Description |
| --- | --- | --- |
| `presentationNodeType` | int32 | — |
| `traitIds` | array&lt;string&gt; | — |
| `traitHashes` | array&lt;uint32&gt; → DestinyTraitDefinition | — |
| `parentNodeHashes` | array&lt;uint32&gt; → DestinyPresentationNodeDefinition | A quick reference to presentation nodes that have this node as a child. Presentation nodes can be parented under multiple parents. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.DestinyPresentationNodeTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Default` | 0 | — |
| `Category` | 1 | — |
| `Collectibles` | 2 | — |
| `Records` | 3 | — |
| `Metric` | 4 | — |
| `Craftable` | 5 | — |

#### Destiny.Definitions.Presentation.DestinyScoredPresentationNodeBaseDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `maxCategoryRecordScore` | int32 | — |
| `presentationNodeType` | int32 | — |
| `traitIds` | array&lt;string&gt; | — |
| `traitHashes` | array&lt;uint32&gt; → DestinyTraitDefinition | — |
| `parentNodeHashes` | array&lt;uint32&gt; → DestinyPresentationNodeDefinition | A quick reference to presentation nodes that have this node as a child. Presentation nodes can be parented under multiple parents. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Presentation.DestinyPresentationNodeDefinition

**Object** · *(Manifest definition, table `PresentationNodes`)*

A PresentationNode is an entity that represents a logical grouping of other entities visually/ organizationally. For now, Presentation Nodes may contain the following... but it may be used for more in the future: - Collectibles - Records (Or, as the public will call them, "Triumphs." Don't ask me why we're overloading the term "Triumph", it still hurts me to think about it) - Metrics (aka Stat Trackers) - Other Presentation Nodes, allowing a tree of Presentation Nodes to be created Part of me wants to break these into conceptual definitions per entity being collected, but the possibility of these different types being mixed in the same UI and the possibility that it could actually be more useful to return the "bare metal" presentation node concept has resulted in me deciding against that for the time being. We'll see if I come to regret this as well.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `originalIcon` | string | The original icon for this presentation node, before we futzed with it. |
| `rootViewIcon` | string | Some presentation nodes are meant to be explicitly shown on the "root" or "entry" screens for the feature to which they are related. You should use this icon when showing them on such a view, if you have a similar "entry point" view in your UI. If you don't have a UI, then I guess it doesn't matter either way does it? |
| `nodeType` | int32 | — |
| `isSeasonal` | boolean | Primarily for Guardian Ranks, this property if the contents of this node are tied to the current season. These nodes are shown with a different color for the in-game Guardian Ranks display. |
| `scope` | int32 | Indicates whether this presentation node's state is determined on a per-character or on an account-wide basis. |
| `objectiveHash` | uint32 → DestinyObjectiveDefinition? | If this presentation node shows a related objective (for instance, if it tracks the progress of its children), the objective being tracked is indicated here. |
| `completionRecordHash` | uint32 → DestinyRecordDefinition? | If this presentation node has an associated "Record" that you can accomplish for completing its children, this is the identifier of that Record. |
| `children` | Destiny.Definitions.Presentation.DestinyPresentationNodeChildrenBlock | The child entities contained by this presentation node. |
| `displayStyle` | int32 | A hint for how to display this presentation node when it's shown in a list. |
| `screenStyle` | int32 | A hint for how to display this presentation node when it's shown in its own detail screen. |
| `requirements` | Destiny.Definitions.Presentation.DestinyPresentationNodeRequirementsBlock | The requirements for being able to interact with this presentation node and its children. |
| `disableChildSubscreenNavigation` | boolean | If this presentation node has children, but the game doesn't let you inspect the details of those children, that is indicated here. |
| `maxCategoryRecordScore` | int32 | — |
| `presentationNodeType` | int32 | — |
| `traitIds` | array&lt;string&gt; | — |
| `traitHashes` | array&lt;uint32&gt; → DestinyTraitDefinition | — |
| `parentNodeHashes` | array&lt;uint32&gt; → DestinyPresentationNodeDefinition | A quick reference to presentation nodes that have this node as a child. Presentation nodes can be parented under multiple parents. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.DestinyScopeEnumeration

**Enum** (`int32`)

There's a lot of places where we need to know scope on more than just a profile or character level. For everything else, there's this more generic sense of scope.

| Value | # | Description |
| --- | --- | --- |
| `Profile` | 0 | — |
| `Character` | 1 | — |

#### Destiny.Definitions.Presentation.DestinyPresentationNodeChildrenBlock

**Type:** object

As/if presentation nodes begin to host more entities as children, these lists will be added to. One list property exists per type of entity that can be treated as a child of this presentation node, and each holds the identifier of the entity and any associated information needed to display the UI for that entity (if anything)

| Property | Type | Description |
| --- | --- | --- |
| `presentationNodes` | array&lt;Destiny.Definitions.Presentation.DestinyPresentationNodeChildEntry&gt; | — |
| `collectibles` | array&lt;Destiny.Definitions.Presentation.DestinyPresentationNodeCollectibleChildEntry&gt; | — |
| `records` | array&lt;Destiny.Definitions.Presentation.DestinyPresentationNodeRecordChildEntry&gt; | — |
| `metrics` | array&lt;Destiny.Definitions.Presentation.DestinyPresentationNodeMetricChildEntry&gt; | — |
| `craftables` | array&lt;Destiny.Definitions.Presentation.DestinyPresentationNodeCraftableChildEntry&gt; | — |

#### Destiny.Definitions.Presentation.DestinyPresentationNodeChildEntryBase

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `nodeDisplayPriority` | uint32 | Use this value to sort the presentation node children in ascending order. |

#### Destiny.Definitions.Presentation.DestinyPresentationNodeChildEntry

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `presentationNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `nodeDisplayPriority` | uint32 | Use this value to sort the presentation node children in ascending order. |

#### Destiny.Definitions.Presentation.DestinyPresentationNodeCollectibleChildEntry

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `collectibleHash` | uint32 → DestinyCollectibleDefinition | — |
| `nodeDisplayPriority` | uint32 | Use this value to sort the presentation node children in ascending order. |

#### Destiny.Definitions.Collectibles.DestinyCollectibleDefinition

**Object** · *(Manifest definition, table `Collectibles`)*

Defines a

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `scope` | int32 | Indicates whether the state of this Collectible is determined on a per-character or on an account- wide basis. |
| `sourceString` | string | A human readable string for a hint about how to acquire the item. |
| `sourceHash` | uint32? | This is a hash identifier we are building on the BNet side in an attempt to let people group collectibles by similar sources. I can't promise that it's going to be 100% accurate, but if the designers were consistent in assigning the same source strings to items with the same sources, it *ought to* be. No promises though. This hash also doesn't relate to an actual definition, just to note: we've got nothing useful other than the source string for this data. |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | — |
| `acquisitionInfo` | Destiny.Definitions.Collectibles.DestinyCollectibleAcquisitionBlock | — |
| `stateInfo` | Destiny.Definitions.Collectibles.DestinyCollectibleStateBlock | — |
| `presentationInfo` | Destiny.Definitions.Presentation.DestinyPresentationChildBlock | — |
| `presentationNodeType` | int32 | — |
| `traitIds` | array&lt;string&gt; | — |
| `traitHashes` | array&lt;uint32&gt; → DestinyTraitDefinition | — |
| `parentNodeHashes` | array&lt;uint32&gt; → DestinyPresentationNodeDefinition | A quick reference to presentation nodes that have this node as a child. Presentation nodes can be parented under multiple parents. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Collectibles.DestinyCollectibleAcquisitionBlock

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `acquireMaterialRequirementHash` | uint32 → DestinyMaterialRequirementSetDefinition? | — |
| `acquireTimestampUnlockValueHash` | uint32 → DestinyUnlockValueDefinition? | — |

#### Destiny.Definitions.DestinyUnlockValueDefinition

**Type:** object

An Unlock Value is an internal integer value, stored on the server and used in a variety of ways, most frequently for the gating/requirement checks that the game performs across all of its main features. They can also be used as the storage data for mapped Progressions, Objectives, and other features that require storage of variable numeric values.

| Property | Type | Description |
| --- | --- | --- |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Collectibles.DestinyCollectibleStateBlock

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `obscuredOverrideItemHash` | uint32 → DestinyInventoryItemDefinition? | — |
| `requirements` | Destiny.Definitions.Presentation.DestinyPresentationNodeRequirementsBlock | — |

#### Destiny.Definitions.Presentation.DestinyPresentationNodeRequirementsBlock

**Type:** object

Presentation nodes can be restricted by various requirements. This defines the rules of those requirements, and the message(s) to be shown if these requirements aren't met.

| Property | Type | Description |
| --- | --- | --- |
| `entitlementUnavailableMessage` | string | If this node is not accessible due to Entitlements (for instance, you don't own the required game expansion), this is the message to show. |

#### Destiny.Definitions.Presentation.DestinyPresentationChildBlock

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `presentationNodeType` | int32 | — |
| `parentPresentationNodeHashes` | array&lt;uint32&gt; → DestinyPresentationNodeDefinition | — |
| `displayStyle` | int32 | — |

#### Destiny.DestinyPresentationDisplayStyleEnumeration

**Enum** (`int32`)

A hint for how the presentation node should be displayed when shown in a list. How you use this is your UI is up to you.

| Value | # | Description |
| --- | --- | --- |
| `Category` | 0 | Display the item as a category, through which sub-items are filtered. |
| `Badge` | 1 | — |
| `Medals` | 2 | — |
| `Collectible` | 3 | — |
| `Record` | 4 | — |
| `SeasonalTriumph` | 5 | — |
| `GuardianRank` | 6 | — |
| `CategoryCollectibles` | 7 | — |
| `CategoryCurrencies` | 8 | — |
| `CategoryEmblems` | 9 | — |
| `CategoryEmotes` | 10 | — |
| `CategoryEngrams` | 11 | — |
| `CategoryFinishers` | 12 | — |
| `CategoryGhosts` | 13 | — |
| `CategoryMisc` | 14 | — |
| `CategoryMods` | 15 | — |
| `CategoryOrnaments` | 16 | — |
| `CategoryShaders` | 17 | — |
| `CategoryShips` | 18 | — |
| `CategorySpawnfx` | 19 | — |
| `CategoryUpgradeMaterials` | 20 | — |

#### Destiny.Definitions.Presentation.DestinyPresentationNodeRecordChildEntry

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `recordHash` | uint32 → DestinyRecordDefinition | — |
| `nodeDisplayPriority` | uint32 | Use this value to sort the presentation node children in ascending order. |

#### Destiny.Definitions.Records.DestinyRecordDefinition

**Object** · *(Manifest definition, table `Records`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `scope` | int32 | Indicates whether this Record's state is determined on a per-character or on an account-wide basis. |
| `presentationInfo` | Destiny.Definitions.Presentation.DestinyPresentationChildBlock | — |
| `loreHash` | uint32 → DestinyLoreDefinition? | — |
| `objectiveHashes` | array&lt;uint32&gt; → DestinyObjectiveDefinition | — |
| `recordValueStyle` | int32 | — |
| `forTitleGilding` | boolean | — |
| `shouldShowLargeIcons` | boolean | A hint to show a large icon for a reward |
| `titleInfo` | Destiny.Definitions.Records.DestinyRecordTitleBlock | — |
| `completionInfo` | Destiny.Definitions.Records.DestinyRecordCompletionBlock | — |
| `stateInfo` | Destiny.Definitions.Records.SchemaRecordStateBlock | — |
| `requirements` | Destiny.Definitions.Presentation.DestinyPresentationNodeRequirementsBlock | — |
| `expirationInfo` | Destiny.Definitions.Records.DestinyRecordExpirationBlock | — |
| `intervalInfo` | Destiny.Definitions.Records.DestinyRecordIntervalBlock | Some records have multiple 'interval' objectives, and the record may be claimed at each completed interval |
| `rewardItems` | array&lt;Destiny.DestinyItemQuantity&gt; | If there is any publicly available information about rewards earned for achieving this record, this is the list of those items. However, note that some records intentionally have "hidden" rewards. These will not be returned in this list. |
| `recordTypeName` | string | A display name for the type of record this is (Triumphs, Lore, Medals, Seasonal Challenge, etc.). |
| `presentationNodeType` | int32 | — |
| `traitIds` | array&lt;string&gt; | — |
| `traitHashes` | array&lt;uint32&gt; → DestinyTraitDefinition | — |
| `parentNodeHashes` | array&lt;uint32&gt; → DestinyPresentationNodeDefinition | A quick reference to presentation nodes that have this node as a child. Presentation nodes can be parented under multiple parents. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.DestinyRecordValueStyleEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Integer` | 0 | — |
| `Percentage` | 1 | — |
| `Milliseconds` | 2 | — |
| `Boolean` | 3 | — |
| `Decimal` | 4 | — |

#### Destiny.Definitions.Records.DestinyRecordTitleBlock

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `hasTitle` | boolean | — |
| `titlesByGender` | Mapping&lt;int32, string&gt; | — |
| `titlesByGenderHash` | Mapping&lt;uint32, string&gt; → DestinyGenderDefinition | For those who prefer to use the definitions. |
| `gildingTrackingRecordHash` | uint32 → DestinyRecordDefinition? | — |

#### Destiny.Definitions.Records.DestinyRecordCompletionBlock

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `partialCompletionObjectiveCountThreshold` | int32 | The number of objectives that must be completed before the objective is considered "complete" |
| `ScoreValue` | int32 | — |
| `shouldFireToast` | boolean | — |
| `toastStyle` | int32 | — |

#### Destiny.DestinyRecordToastStyleEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Record` | 1 | — |
| `Lore` | 2 | — |
| `Badge` | 3 | — |
| `MetaRecord` | 4 | — |
| `MedalComplete` | 5 | — |
| `SeasonChallengeComplete` | 6 | — |
| `GildedTitleComplete` | 7 | — |
| `CraftingRecipeUnlocked` | 8 | — |
| `ToastGuardianRankDetails` | 9 | — |
| `PathfinderObjectiveCompleteRituals` | 10 | — |
| `PathfinderObjectiveCompleteSchism` | 11 | — |
| `PathfinderObjectiveCompletePvp` | 12 | — |
| `PathfinderObjectiveCompleteStrikes` | 13 | — |
| `PathfinderObjectiveCompleteGambit` | 14 | — |
| `SeasonWeeklyComplete` | 15 | — |
| `SeasonDailyComplete` | 16 | — |

#### Destiny.Definitions.Records.SchemaRecordStateBlock

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `featuredPriority` | int32 | — |
| `obscuredName` | string | A display name override to show when this record is 'obscured' instead of the default obscured display name. |
| `obscuredDescription` | string | A display description override to show when this record is 'obscured' instead of the default obscured display description. |

#### Destiny.Definitions.Records.DestinyRecordExpirationBlock

**Type:** object

If this record has an expiration after which it cannot be earned, this is some information about that expiration.

| Property | Type | Description |
| --- | --- | --- |
| `hasExpiration` | boolean | — |
| `description` | string | — |
| `icon` | string | — |

#### Destiny.Definitions.Records.DestinyRecordIntervalBlock

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `intervalObjectives` | array&lt;Destiny.Definitions.Records.DestinyRecordIntervalObjective&gt; | — |
| `intervalRewards` | array&lt;Destiny.Definitions.Records.DestinyRecordIntervalRewards&gt; | — |
| `originalObjectiveArrayInsertionIndex` | int32 | — |

#### Destiny.Definitions.Records.DestinyRecordIntervalObjective

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `intervalObjectiveHash` | uint32 → DestinyObjectiveDefinition | — |
| `intervalScoreValue` | int32 | — |

#### Destiny.Definitions.Records.DestinyRecordIntervalRewards

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `intervalRewardItems` | array&lt;Destiny.DestinyItemQuantity&gt; | — |

#### Destiny.Definitions.Lore.DestinyLoreDefinition

**Object** · *(Manifest definition, table `Lore`)*

These are definitions for in-game "Lore," meant to be narrative enhancements of the game experience. DestinyInventoryItemDefinitions for interesting items point to these definitions, but nothing's stopping you from scraping all of these and doing something cool with them. If they end up having cool data.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `subtitle` | string | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Presentation.DestinyPresentationNodeMetricChildEntry

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `metricHash` | uint32 → DestinyMetricDefinition | — |
| `nodeDisplayPriority` | uint32 | Use this value to sort the presentation node children in ascending order. |

#### Destiny.Definitions.Metrics.DestinyMetricDefinition

**Object** · *(Manifest definition, table `Metrics`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `trackingObjectiveHash` | uint32 → DestinyObjectiveDefinition | — |
| `lowerValueIsBetter` | boolean | — |
| `presentationNodeType` | int32 | — |
| `traitIds` | array&lt;string&gt; | — |
| `traitHashes` | array&lt;uint32&gt; → DestinyTraitDefinition | — |
| `parentNodeHashes` | array&lt;uint32&gt; → DestinyPresentationNodeDefinition | A quick reference to presentation nodes that have this node as a child. Presentation nodes can be parented under multiple parents. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Presentation.DestinyPresentationNodeCraftableChildEntry

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `craftableItemHash` | uint32 → DestinyInventoryItemDefinition | — |
| `nodeDisplayPriority` | uint32 | Use this value to sort the presentation node children in ascending order. |

#### Destiny.DestinyPresentationScreenStyleEnumeration

**Enum** (`int32`)

A hint for what screen should be shown when this presentation node is clicked into. How you use this is your UI is up to you.

| Value | # | Description |
| --- | --- | --- |
| `Default` | 0 | Use the "default" view for the presentation nodes. |
| `CategorySets` | 1 | Show sub-items as "category sets". In-game, you'd see these as a vertical list of child presentation nodes - armor sets for example - and the icons of items within those sets displayed horizontally. |
| `Badge` | 2 | Show sub-items as Badges. (I know, I know. We don't need no stinkin' badges har har har) |

#### Destiny.Definitions.Items.DestinyItemPlugDefinition

**Type:** object

If an item is a Plug, its DestinyInventoryItemDefinition.plug property will be populated with an instance of one of these bad boys. This gives information about when it can be inserted, what the plug's category is (and thus whether it is compatible with a socket... see DestinySocketTypeDefinition for information about Plug Categories and socket compatibility), whether it is enabled and other Plug info.

| Property | Type | Description |
| --- | --- | --- |
| `insertionRules` | array&lt;Destiny.Definitions.Items.DestinyPlugRuleDefinition&gt; | The rules around when this plug can be inserted into a socket, aside from the socket's individual restrictions. The live data DestinyItemPlugComponent.insertFailIndexes will be an index into this array, so you can pull out the failure strings appropriate for the user. |
| `plugCategoryIdentifier` | string | The string identifier for the plug's category. Use the socket's DestinySocketTypeDefinition.plugWhitelist to determine whether this plug can be inserted into the socket. |
| `plugCategoryHash` | uint32 | The hash for the plugCategoryIdentifier. You can use this instead if you wish: I put both in the definition for debugging purposes. |
| `onActionRecreateSelf` | boolean | If you successfully socket the item, this will determine whether or not you get "refunded" on the plug. |
| `insertionMaterialRequirementHash` | uint32 → DestinyMaterialRequirementSetDefinition | If inserting this plug requires materials, this is the hash identifier for looking up the DestinyMaterialRequirementSetDefinition for those requirements. |
| `previewItemOverrideHash` | uint32 → DestinyInventoryItemDefinition | In the game, if you're inspecting a plug item directly, this will be the item shown with the plug attached. Look up the DestinyInventoryItemDefinition for this hash for the item. |
| `enabledMaterialRequirementHash` | uint32 → DestinyMaterialRequirementSetDefinition | It's not enough for the plug to be inserted. It has to be enabled as well. For it to be enabled, it may require materials. This is the hash identifier for the DestinyMaterialRequirementSetDefinition for those requirements, if there is one. |
| `enabledRules` | array&lt;Destiny.Definitions.Items.DestinyPlugRuleDefinition&gt; | The rules around whether the plug, once inserted, is enabled and providing its benefits. The live data DestinyItemPlugComponent.enableFailIndexes will be an index into this array, so you can pull out the failure strings appropriate for the user. |
| `uiPlugLabel` | string | Plugs can have arbitrary, UI-defined identifiers that the UI designers use to determine the style applied to plugs. Unfortunately, we have neither a definitive list of these labels nor advance warning of when new labels might be applied or how that relates to how they get rendered. If you want to, you can refer to known labels to change your own styles: but know that new ones can be created arbitrarily, and we have no way of associating the labels with any specific UI style guidance... you'll have to piece that together on your end. Or do what we do, and just show plugs more generically, without specialized styles. |
| `plugStyle` | int32 | — |
| `plugAvailability` | int32 | Indicates the rules about when this plug can be used. See the PlugAvailabilityMode enumeration for more information! |
| `alternateUiPlugLabel` | string | If the plug meets certain state requirements, it may have an alternative label applied to it. This is the alternative label that will be applied in such a situation. |
| `alternatePlugStyle` | int32 | The alternate plug of the plug: only applies when the item is in states that only the server can know about and control, unfortunately. See AlternateUiPlugLabel for the related label info. |
| `isDummyPlug` | boolean | If TRUE, this plug is used for UI display purposes only, and doesn't have any interesting effects of its own. |
| `parentItemOverride` | Destiny.Definitions.Items.DestinyParentItemOverride | Do you ever get the feeling that a system has become so overburdened by edge cases that it probably should have become some other system entirely? So do I! In totally unrelated news, Plugs can now override properties of their parent items. This is some of the relevant definition data for those overrides. If this is populated, it will have the override data to be applied when this plug is applied to an item. |
| `energyCapacity` | Destiny.Definitions.Items.DestinyEnergyCapacityEntry | IF not null, this plug provides Energy capacity to the item in which it is socketed. In Armor 2.0 for example, is implemented in a similar way to Masterworks, where visually it's a single area of the UI being clicked on to "Upgrade" to higher energy levels, but it's actually socketing new plugs. |
| `energyCost` | Destiny.Definitions.Items.DestinyEnergyCostEntry | IF not null, this plug has an energy cost. This contains the details of that cost. |

#### Destiny.Definitions.Items.DestinyPlugRuleDefinition

**Type:** object

Dictates a rule around whether the plug is enabled or insertable. In practice, the live Destiny data will refer to these entries by index. You can then look up that index in the appropriate property (enabledRules or insertionRules) to get the localized string for the failure message if it failed.

| Property | Type | Description |
| --- | --- | --- |
| `failureMessage` | string | The localized string to show if this rule fails. |

#### Destiny.PlugUiStylesEnumeration

**Enum** (`int32`)

If the plug has a specific custom style, this enumeration will represent that style/those styles.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Masterwork` | 1 | — |

#### Destiny.PlugAvailabilityModeEnumeration

**Enum** (`int32`)

This enum determines whether the plug is available to be inserted. - Normal means that all existing rules for plug insertion apply. - UnavailableIfSocketContainsMatchingPlugCategory means that the plug is only available if the socket does NOT match the plug category. - AvailableIfSocketContainsMatchingPlugCategory means that the plug is only available if the socket DOES match the plug category. For category matching, use the plug's "plugCategoryIdentifier" property, comparing it to

| Value | # | Description |
| --- | --- | --- |
| `Normal` | 0 | — |
| `UnavailableIfSocketContainsMatchingPlugCategory` | 1 | — |
| `AvailableIfSocketContainsMatchingPlugCategory` | 2 | — |

#### Destiny.Definitions.Items.DestinyParentItemOverride

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `additionalEquipRequirementsDisplayStrings` | array&lt;string&gt; | — |
| `pipIcon` | string | — |

#### Destiny.Definitions.Items.DestinyEnergyCapacityEntry

**Type:** object

Items can have Energy Capacity, and plugs can provide that capacity such as on a piece of Armor in Armor 2.0. This is how much "Energy" can be spent on activating plugs for this item.

| Property | Type | Description |
| --- | --- | --- |
| `capacityValue` | int32 | How much energy capacity this plug provides. |
| `energyTypeHash` | uint32 → DestinyEnergyTypeDefinition | Energy provided by a plug is always of a specific type - this is the hash identifier for the energy type for which it provides Capacity. |
| `energyType` | int32 | The Energy Type for this energy capacity, in enum form for easy use. |

#### Destiny.DestinyEnergyTypeEnumeration

**Enum** (`int32`)

Represents the socket energy types for Armor 2.0, Ghosts 2.0, and Stasis subclasses.

| Value | # | Description |
| --- | --- | --- |
| `Any` | 0 | — |
| `Arc` | 1 | — |
| `Thermal` | 2 | — |
| `Void` | 3 | — |
| `Ghost` | 4 | — |
| `Subclass` | 5 | — |
| `Stasis` | 6 | — |

#### Destiny.Definitions.EnergyTypes.DestinyEnergyTypeDefinition

**Object** · *(Manifest definition, table `EnergyTypes`)*

Represents types of Energy that can be used for costs and payments related to Armor 2.0 mods.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | The description of the energy type, icon etc... |
| `transparentIconPath` | string | A variant of the icon that is transparent and colorless. |
| `showIcon` | boolean | If TRUE, the game shows this Energy type's icon. Otherwise, it doesn't. Whether you show it or not is up to you. |
| `enumValue` | int32 | We have an enumeration for Energy types for quick reference. This is the current definition's Energy type enum value. |
| `capacityStatHash` | uint32 → DestinyStatDefinition? | If this Energy Type can be used for determining the Type of Energy that an item can consume, this is the hash for the DestinyInvestmentStatDefinition that represents the stat which holds the Capacity for that energy type. (Note that this is optional because "Any" is a valid cost, but not valid for Capacity - an Armor must have a specific Energy Type for determining the energy type that the Armor is restricted to use) |
| `costStatHash` | uint32 → DestinyStatDefinition | If this Energy Type can be used as a cost to pay for socketing Armor 2.0 items, this is the hash for the DestinyInvestmentStatDefinition that stores the plug's raw cost. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Items.DestinyEnergyCostEntry

**Type:** object

Some plugs cost Energy, which is a stat on the item that can be increased by other plugs (that, at least in Armor 2.0, have a "masterworks-like" mechanic for upgrading). If a plug has costs, the details of that cost are defined here.

| Property | Type | Description |
| --- | --- | --- |
| `energyCost` | int32 | The Energy cost for inserting this plug. |
| `energyTypeHash` | uint32 → DestinyEnergyTypeDefinition | The type of energy that this plug costs, as a reference to the DestinyEnergyTypeDefinition of the energy type. |
| `energyType` | int32 | The type of energy that this plug costs, in enum form. |

#### Destiny.Definitions.DestinyItemGearsetBlockDefinition

**Type:** object

If an item has a related gearset, this is the list of items in that set, and an unlock expression that evaluates to a number representing the progress toward gearset completion (a very rare use for unlock expressions!)

| Property | Type | Description |
| --- | --- | --- |
| `trackingValueMax` | int32 | The maximum possible number of items that can be collected. |
| `itemList` | array&lt;uint32&gt; → DestinyInventoryItemDefinition | The list of hashes for items in the gearset. Use them to look up DestinyInventoryItemDefinition entries for the items in the set. |

#### Destiny.Definitions.DestinyItemSackBlockDefinition

**Type:** object

Some items are "sacks" - they can be "opened" to produce other items. This is information related to its sack status, mostly UI strings. Engrams are an example of items that are considered to be "Sacks".

| Property | Type | Description |
| --- | --- | --- |
| `detailAction` | string | A description of what will happen when you open the sack. As far as I can tell, this is blank currently. Unknown whether it will eventually be populated with useful info. |
| `openAction` | string | The localized name of the action being performed when you open the sack. |
| `selectItemCount` | int32 | — |
| `vendorSackType` | string | — |
| `openOnAcquire` | boolean | — |

#### Destiny.Definitions.DestinyItemSocketBlockDefinition

**Type:** object

If defined, the item has at least one socket.

| Property | Type | Description |
| --- | --- | --- |
| `detail` | string | This was supposed to be a string that would give per-item details about sockets. In practice, it turns out that all this ever has is the localized word "details". ... that's lame, but perhaps it will become something cool in the future. |
| `socketEntries` | array&lt;Destiny.Definitions.DestinyItemSocketEntryDefinition&gt; | Each non-intrinsic (or mutable) socket on an item is defined here. Check inside for more info. |
| `intrinsicSockets` | array&lt;Destiny.Definitions.DestinyItemIntrinsicSocketEntryDefinition&gt; | Each intrinsic (or immutable/permanent) socket on an item is defined here, along with the plug that is permanently affixed to the socket. |
| `socketCategories` | array&lt;Destiny.Definitions.DestinyItemSocketCategoryDefinition&gt; | A convenience property, that refers to the sockets in the "sockets" property, pre-grouped by category and ordered in the manner that they should be grouped in the UI. You could form this yourself with the existing data, but why would you want to? Enjoy life man. |

#### Destiny.Definitions.DestinyItemSocketEntryDefinition

**Type:** object

The definition information for a specific socket on an item. This will determine how the socket behaves in-game.

| Property | Type | Description |
| --- | --- | --- |
| `socketTypeHash` | uint32 → DestinySocketTypeDefinition | All sockets have a type, and this is the hash identifier for this particular type. Use it to look up the DestinySocketTypeDefinition: read there for more information on how socket types affect the behavior of the socket. |
| `singleInitialItemHash` | uint32 → DestinyInventoryItemDefinition | If a valid hash, this is the hash identifier for the DestinyInventoryItemDefinition representing the Plug that will be initially inserted into the item on item creation. Otherwise, this Socket will either start without a plug inserted, or will have one randomly inserted. |
| `reusablePlugItems` | array&lt;Destiny.Definitions.DestinyItemSocketEntryPlugItemDefinition&gt; | This is a list of pre-determined plugs that can *always* be plugged into this socket, without the character having the plug in their inventory. If this list is populated, you will not be allowed to plug an arbitrary item in the socket: you will only be able to choose from one of these reusable plugs. |
| `preventInitializationOnVendorPurchase` | boolean | If this is true, then the socket will not be initialized with a plug if the item is purchased from a Vendor. Remember that Vendors are much more than conceptual vendors: they include "Collection Kiosks" and other entities. See DestinyVendorDefinition for more information. |
| `hidePerksInItemTooltip` | boolean | If this is true, the perks provided by this socket shouldn't be shown in the item's tooltip. This might be useful if it's providing a hidden bonus, or if the bonus is less important than other benefits on the item. |
| `plugSources` | int32 | Indicates where you should go to get plugs for this socket. This will affect how you populate your UI, as well as what plugs are valid for this socket. It's an alternative to having to check for the existence of certain properties (reusablePlugItems for example) to infer where plugs should come from. |
| `reusablePlugSetHash` | uint32 → DestinyPlugSetDefinition? | If this socket's plugs come from a reusable DestinyPlugSetDefinition, this is the identifier for that set. We added this concept to reduce some major duplication that's going to come from sockets as replacements for what was once implemented as large sets of items and kiosks (like Emotes). As of Shadowkeep, these will come up much more frequently and be driven by game content rather than custom curation. |
| `randomizedPlugSetHash` | uint32 → DestinyPlugSetDefinition? | This field replaces "randomizedPlugItems" as of Shadowkeep launch. If a socket has randomized plugs, this is a pointer to the set of plugs that could be used, as defined in DestinyPlugSetDefinition. If null, the item has no randomized plugs. |
| `defaultVisible` | boolean | If true, then this socket is visible in the item's "default" state. If you have an instance, you should always check the runtime state, as that can override this visibility setting: but if you're looking at the item on a conceptual level, this property can be useful for hiding data such as legacy sockets - which remain defined on items for infrastructure purposes, but can be confusing for users to see. |

#### Destiny.Definitions.DestinyItemSocketEntryPlugItemDefinition

**Type:** object

The definition of a known, reusable plug that can be applied to a socket.

| Property | Type | Description |
| --- | --- | --- |
| `plugItemHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier of a DestinyInventoryItemDefinition representing the plug that can be inserted. |

#### Destiny.SocketPlugSourcesEnumeration

**Enum** (`int32`)

Indicates how a socket is populated, and where you should look for valid plug data. This is a flags enumeration/bitmask field, as you may have to look in multiple sources across multiple components for valid plugs. For instance, a socket could have plugs that are sourced from its own definition, as well as plugs that are sourced from Character-scoped AND profile-scoped Plug Sets. Only by combining plug data for every indicated source will you be able to know all of the plugs available for a socket.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | If there's no way we can detect to insert new plugs. |
| `InventorySourced` | 1 | Use plugs found in the player's inventory, based on the socket type rules (see DestinySocketTypeDefinition for more info) Note that a socket - like Shaders - can have *both* reusable plugs and inventory items inserted theoretically. |
| `ReusablePlugItems` | 2 | Use the DestinyItemSocketsComponent.sockets.reusablePlugs property to determine which plugs are valid for this socket. This may have to be combined with other sources, such as plug sets, if those flags are set. Note that "Reusable" plugs may not necessarily come from a plug set, nor from the "reusablePlugItems" in the socket's Definition data. They can sometimes be "randomized" in which case the only source of truth at the moment is still the runtime DestinyItemSocketsComponent.sockets.reusablePlugs property. |
| `ProfilePlugSet` | 4 | Use the ProfilePlugSets (DestinyProfileResponse.profilePlugSets) component data to determine which plugs are valid for this socket. |
| `CharacterPlugSet` | 8 | Use the CharacterPlugSets (DestinyProfileResponse.characterPlugSets) component data to determine which plugs are valid for this socket. |

#### Destiny.Definitions.Sockets.DestinyPlugSetDefinition

**Object** · *(Manifest definition, table `PlugSets`)*

Sometimes, we have large sets of reusable plugs that are defined identically and thus can (and in some cases, are so large that they *must*) be shared across the places where they are used. These are the definitions for those reusable sets of plugs. See DestinyItemSocketEntryDefinition.plugSource and reusablePlugSetHash for the relationship between these reusable plug sets and the sockets that leverage them (for starters, Emotes). As of the release of Shadowkeep (Late 2019), these will begin to be sourced from game content directly - which means there will be many more of them, but it also means we may not get all data that we used to get for them. DisplayProperties, in particular, will no longer be guaranteed to contain valid information. We will make a best effort to guess what ought to be populated there where possible, but it will be invalid for many/ most plug sets.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | If you want to show these plugs in isolation, these are the display properties for them. |
| `reusablePlugItems` | array&lt;Destiny.Definitions.DestinyItemSocketEntryPlugItemRandomizedDefinition&gt; | This is a list of pre-determined plugs that can be plugged into this socket, without the character having the plug in their inventory. If this list is populated, you will not be allowed to plug an arbitrary item in the socket: you will only be able to choose from one of these reusable plugs. |
| `isFakePlugSet` | boolean | Mostly for our debugging or reporting bugs, BNet is making "fake" plug sets in a desperate effort to reduce socket sizes. If this is true, the plug set was generated by BNet: if it looks wrong, that's a good indicator that it's bungie.net that fucked this up. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyItemSocketEntryPlugItemRandomizedDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `craftingRequirements` | Destiny.Definitions.DestinyPlugItemCraftingRequirements | — |
| `currentlyCanRoll` | boolean | Indicates if the plug can be rolled on the current version of the item. For example, older versions of weapons may have plug rolls that are no longer possible on the current versions. |
| `plugItemHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier of a DestinyInventoryItemDefinition representing the plug that can be inserted. |

#### Destiny.Definitions.DestinyPlugItemCraftingRequirements

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `unlockRequirements` | array&lt;Destiny.Definitions.DestinyPlugItemCraftingUnlockRequirement&gt; | — |
| `requiredLevel` | int32? | If the plug has a known level requirement, it'll be available here. |
| `materialRequirementHashes` | array&lt;uint32&gt; → DestinyMaterialRequirementSetDefinition | — |

#### Destiny.Definitions.DestinyPlugItemCraftingUnlockRequirement

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `failureDescription` | string | — |

#### Destiny.Definitions.DestinyItemIntrinsicSocketEntryDefinition

**Type:** object

Represents a socket that has a plug associated with it intrinsically. This is useful for situations where the weapon needs to have a visual plug/Mod on it, but that plug/Mod should never change.

| Property | Type | Description |
| --- | --- | --- |
| `plugItemHash` | uint32 → DestinyInventoryItemDefinition | Indicates the plug that is intrinsically inserted into this socket. |
| `socketTypeHash` | uint32 → DestinySocketTypeDefinition | Indicates the type of this intrinsic socket. |
| `defaultVisible` | boolean | If true, then this socket is visible in the item's "default" state. If you have an instance, you should always check the runtime state, as that can override this visibility setting: but if you're looking at the item on a conceptual level, this property can be useful for hiding data such as legacy sockets - which remain defined on items for infrastructure purposes, but can be confusing for users to see. |

#### Destiny.Definitions.DestinyItemSocketCategoryDefinition

**Type:** object

Sockets are grouped into categories in the UI. These define which category and which sockets are under that category.

| Property | Type | Description |
| --- | --- | --- |
| `socketCategoryHash` | uint32 → DestinySocketCategoryDefinition | The hash for the Socket Category: a quick way to go get the header display information for the category. Use it to look up DestinySocketCategoryDefinition info. |
| `socketIndexes` | array&lt;int32&gt; | Use these indexes to look up the sockets in the "sockets.socketEntries" property on the item definition. These are the indexes under the category, in game-rendered order. |

#### Destiny.Definitions.DestinyItemSummaryBlockDefinition

**Type:** object

This appears to be information used when rendering rewards. We don't currently use it on BNet.

| Property | Type | Description |
| --- | --- | --- |
| `sortPriority` | int32 | Apparently when rendering an item in a reward, this should be used as a sort priority. We're not doing it presently. |

#### Destiny.Definitions.DestinyItemTalentGridBlockDefinition

**Type:** object

This defines information that can only come from a talent grid on an item. Items mostly have negligible talent grid data these days, but instanced items still retain grids as a source for some of this common information. Builds/Subclasses are the only items left that still have talent grids with meaningful Nodes.

| Property | Type | Description |
| --- | --- | --- |
| `talentGridHash` | uint32 → DestinyTalentGridDefinition | The hash identifier of the DestinyTalentGridDefinition attached to this item. |
| `itemDetailString` | string | This is meant to be a subtitle for looking at the talent grid. In practice, somewhat frustratingly, this always merely says the localized word for "Details". Great. Maybe it'll have more if talent grids ever get used for more than builds and subclasses again. |
| `buildName` | string | A shortcut string identifier for the "build" in question, if this talent grid has an associated build. Doesn't map to anything we can expose at the moment. |
| `hudDamageType` | int32 | If the talent grid implies a damage type, this is the enum value for that damage type. |
| `hudIcon` | string | If the talent grid has a special icon that's shown in the game UI (like builds, funny that), this is the identifier for that icon. Sadly, we don't actually get that icon right now. I'll be looking to replace this with a path to the actual icon itself. |

#### Destiny.Definitions.DestinyTalentGridDefinition

**Object** · *(Manifest definition, table `Talents`)*

The time has unfortunately come to talk about Talent Grids. Talent Grids are the most complex and unintuitive part of the Destiny Definition data. Grab a cup of coffee before we begin, I can wait. Talent Grids were the primary way that items could be customized in Destiny 1. In Destiny 2, for now, talent grids have become exclusively used by Subclass/Build items: but the system is still in place for it to be used by items should the direction change back toward talent grids. Talent Grids have Nodes: the visual circles on the talent grid detail screen that have icons and can be activated if you meet certain requirements and pay costs. The actual visual data and effects, however, are driven by the "Steps" on Talent Nodes. Any given node will have 1:M of these steps, and the specific step that will be considered the "current" step (and thus the dictator of all benefits, visual state, and activation requirements on the Node) will almost always not be determined until an instance of the item is created. This is how, in Destiny 1, items were able to have such a wide variety of what users saw as "Perks": they were actually Talent Grids with nodes that had a wide variety of Steps, randomly chosen at the time of item creation. Now that Talent Grids are used exclusively by subclasses and builds, all of the properties within still apply: but there are additional visual elements on the Subclass/Build screens that are superimposed on top of the talent nodes. Unfortunately, BNet doesn't have this data: if you want to build a subclass screen, you will have to provide your own "decorative" assets, such as the visual connectors between nodes and the fancy colored-fire-bathed character standing behind the nodes. DestinyInventoryItem.talentGrid.talentGridHash defines an item's linked Talent Grid, which brings you to this definition that contains enough satic data about talent grids to make your head spin. These *must* be combined with instanced data - found when live data returns DestinyItemTalentGridComponent - in order to derive meaning. The instanced data will reference nodes and steps within these definitions, which you will then have to look up in the definition and combine with the instanced data to give the user the visual representation of their item's talent grid.

| Property | Type | Description |
| --- | --- | --- |
| `maxGridLevel` | int32 | The maximum possible level of the Talent Grid: at this level, any nodes are allowed to be activated. |
| `gridLevelPerColumn` | int32 | The meaning of this has been lost in the sands of time: it still exists as a property, but appears to be unused in the modern UI of talent grids. It used to imply that each visual "column" of talent nodes required identical progression levels in order to be activated. Returning this value in case it is still useful to someone? Perhaps it's just a bit of interesting history. |
| `progressionHash` | uint32 → DestinyProgressionDefinition | The hash identifier of the Progression (DestinyProgressionDefinition) that drives whether and when Talent Nodes can be activated on the Grid. Items will have instances of this Progression, and will gain experience that will eventually cause the grid to increase in level. As the grid's level increases, it will cross the threshold where nodes can be activated. See DestinyTalentGridStepDefinition's activation requirements for more information. |
| `nodes` | array&lt;Destiny.Definitions.DestinyTalentNodeDefinition&gt; | The list of Talent Nodes on the Grid (recall that Nodes themselves are really just locations in the UI to show whatever their current Step is. You will only know the current step for a node by retrieving instanced data through platform calls to the API that return DestinyItemTalentGridComponent). |
| `exclusiveSets` | array&lt;Destiny.Definitions.DestinyTalentNodeExclusiveSetDefinition&gt; | Talent Nodes can exist in "exclusive sets": these are sets of nodes in which only a single node in the set can be activated at any given time. Activating a node in this set will automatically deactivate the other nodes in the set (referred to as a "Swap"). If a node in the exclusive set has already been activated, the game will not charge you materials to activate another node in the set, even if you have never activated it before, because you already paid the cost to activate one node in the set. Not to be confused with Exclusive Groups. (how the heck do we NOT get confused by that? Jeez) See the groups property for information about that only-tangentially-related concept. |
| `independentNodeIndexes` | array&lt;int32&gt; | This is a quick reference to the indexes of nodes that are not part of exclusive sets. Handy for knowing which talent nodes can only be activated directly, rather than via swapping. |
| `groups` | Mapping&lt;uint32, Destiny.Definitions.DestinyTalentExclusiveGroup&gt; | Talent Nodes can have "Exclusive Groups". These are not to be confused with Exclusive Sets (see exclusiveSets property). Look at the definition of DestinyTalentExclusiveGroup for more information and how they work. These groups are keyed by the "groupHash" from DestinyTalentExclusiveGroup. |
| `nodeCategories` | array&lt;Destiny.Definitions.DestinyTalentNodeCategory&gt; | BNet wants to show talent nodes grouped by similar purpose with localized titles. This is the ordered list of those categories: if you want to show nodes by category, you can iterate over this list, render the displayProperties for the category as the title, and then iterate over the talent nodes referenced by the category to show the related nodes. Note that this is different from Exclusive Groups or Sets, because these categories also incorporate "Independent" nodes that belong to neither sets nor groups. These are purely for visual grouping of nodes rather than functional grouping. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.DestinyTalentNodeDefinition

**Type:** object

Talent Grids on items have Nodes. These nodes have positions in the talent grid's UI, and contain "Steps" (DestinyTalentNodeStepDefinition), one of whom will be the "Current" step. The Current Step determines the visual properties of the node, as well as what the node grants when it is activated. See DestinyTalentGridDefinition for a more complete overview of how Talent Grids work, and how they are used in Destiny 2 (and how they were used in Destiny 1).

| Property | Type | Description |
| --- | --- | --- |
| `nodeIndex` | int32 | The index into the DestinyTalentGridDefinition's "nodes" property where this node is located. Used to uniquely identify the node within the Talent Grid. Note that this is content version dependent: make sure you have the latest version of content before trying to use these properties. |
| `nodeHash` | uint32 | The hash identifier for the node, which unfortunately is also content version dependent but can be (and ideally, should be) used instead of the nodeIndex to uniquely identify the node. The two exist side-by-side for backcompat reasons due to the Great Talent Node Restructuring of Destiny 1, and I ran out of time to remove one of them and standardize on the other. Sorry! |
| `row` | int32 | The visual "row" where the node should be shown in the UI. If negative, then the node is hidden. |
| `column` | int32 | The visual "column" where the node should be shown in the UI. If negative, the node is hidden. |
| `prerequisiteNodeIndexes` | array&lt;int32&gt; | Indexes into the DestinyTalentGridDefinition.nodes property for any nodes that must be activated before this one is allowed to be activated. I would have liked to change this to hashes for Destiny 2, but we have run out of time. |
| `binaryPairNodeIndex` | int32 | At one point, Talent Nodes supported the idea of "Binary Pairs": nodes that overlapped each other visually, and where activating one deactivated the other. They ended up not being used, mostly because Exclusive Sets are *almost* a superset of this concept, but the potential for it to be used still exists in theory. If this is ever used, this will be the index into the DestinyTalentGridDefinition.nodes property for the node that is the binary pair match to this node. Activating one deactivates the other. |
| `autoUnlocks` | boolean | If true, this node will automatically unlock when the Talent Grid's level reaches the required level of the current step of this node. |
| `lastStepRepeats` | boolean | At one point, Nodes were going to be able to be activated multiple times, changing the current step and potentially piling on multiple effects from the previously activated steps. This property would indicate if the last step could be activated multiple times. This is not currently used, but it isn't out of the question that this could end up being used again in a theoretical future. |
| `isRandom` | boolean | If this is true, the node's step is determined randomly rather than the first step being chosen. |
| `randomActivationRequirement` | Destiny.Definitions.DestinyNodeActivationRequirement | At one point, you were going to be able to repurchase talent nodes that had random steps, to "re- roll" the current step of the node (and thus change the properties of your item). This was to be the activation requirement for performing that re-roll. The system still exists to do this, as far as I know, so it may yet come back around! |
| `isRandomRepurchasable` | boolean | If this is true, the node can be "re-rolled" to acquire a different random current step. This is not used, but still exists for a theoretical future of talent grids. |
| `steps` | array&lt;Destiny.Definitions.DestinyNodeStepDefinition&gt; | At this point, "steps" have been obfuscated into conceptual entities, aggregating the underlying notions of "properties" and "true steps". If you need to know a step as it truly exists - such as when recreating Node logic when processing Vendor data - you'll have to use the "realSteps" property below. |
| `exclusiveWithNodeHashes` | array&lt;uint32&gt; | The nodeHash values for nodes that are in an Exclusive Set with this node. See DestinyTalentGridDefinition.exclusiveSets for more info about exclusive sets. Again, note that these are nodeHashes and *not* nodeIndexes. |
| `randomStartProgressionBarAtProgression` | int32 | If the node's step is randomly selected, this is the amount of the Talent Grid's progression experience at which the progression bar for the node should be shown. |
| `layoutIdentifier` | string | A string identifier for a custom visual layout to apply to this talent node. Unfortunately, we do not have any data for rendering these custom layouts. It will be up to you to interpret these strings and change your UI if you want to have custom UI matching these layouts. |
| `groupHash` | uint32? | As of Destiny 2, nodes can exist as part of "Exclusive Groups". These differ from exclusive sets in that, within the group, many nodes can be activated. But the act of activating any node in the group will cause "opposing" nodes (nodes in groups that are not allowed to be activated at the same time as this group) to deactivate. See DestinyTalentExclusiveGroup for more information on the details. This is an identifier for this node's group, if it is part of one. |
| `loreHash` | uint32 → DestinyLoreDefinition? | Talent nodes can be associated with a piece of Lore, generally rendered in a tooltip. This is the hash identifier of the lore element to show, if there is one to be show. |
| `nodeStyleIdentifier` | string | Comes from the talent grid node style: this identifier should be used to determine how to render the node in the UI. |
| `ignoreForCompletion` | boolean | Comes from the talent grid node style: if true, then this node should be ignored for determining whether the grid is complete. |

#### Destiny.Definitions.DestinyNodeActivationRequirement

**Type:** object

Talent nodes have requirements that must be met before they can be activated. This describes the material costs, the Level of the Talent Grid's progression required, and other conditional information that limits whether a talent node can be activated.

| Property | Type | Description |
| --- | --- | --- |
| `gridLevel` | int32 | The Progression level on the Talent Grid required to activate this node. See DestinyTalentGridDefinition.progressionHash for the related Progression, and read DestinyProgressionDefinition's documentation to learn more about Progressions. |
| `materialRequirementHashes` | array&lt;uint32&gt; → DestinyMaterialRequirementSetDefinition | The list of hash identifiers for material requirement sets: materials that are required for the node to be activated. See DestinyMaterialRequirementSetDefinition for more information about material requirements. In this case, only a single DestinyMaterialRequirementSetDefinition will be chosen from this list, and we won't know which one will be chosen until an instance of the item is created. |

#### Destiny.Definitions.DestinyNodeStepDefinition

**Type:** object

This defines the properties of a "Talent Node Step". When you see a talent node in game, the actual visible properties that you see (its icon, description, the perks and stats it provides) are not provided by the Node itself, but rather by the currently active Step on the node. When a Talent Node is activated, the currently active step's benefits are conferred upon the item and character. The currently active step on talent nodes are determined when an item is first instantiated. Sometimes it is random, sometimes it is more deterministic (particularly when a node has only a single step). Note that, when dealing with Talent Node Steps, you must ensure that you have the latest version of content. stepIndex and nodeStepHash - two ways of identifying the step within a node - are both content version dependent, and thus are subject to change between content updates.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | These are the display properties actually used to render the Talent Node. The currently active step's displayProperties are shown. |
| `stepIndex` | int32 | The index of this step in the list of Steps on the Talent Node. Unfortunately, this is the closest thing we have to an identifier for the Step: steps are not provided a content version agnostic identifier. This means that, when you are dealing with talent nodes, you will need to first ensure that you have the latest version of content. |
| `nodeStepHash` | uint32 | The hash of this node step. Unfortunately, while it can be used to uniquely identify the step within a node, it is also content version dependent and should not be relied on without ensuring you have the latest vesion of content. |
| `interactionDescription` | string | If you can interact with this node in some way, this is the localized description of that interaction. |
| `damageType` | int32 | An enum representing a damage type granted by activating this step, if any. |
| `damageTypeHash` | uint32 → DestinyDamageTypeDefinition? | If the step provides a damage type, this will be the hash identifier used to look up the damage type's DestinyDamageTypeDefinition. |
| `activationRequirement` | Destiny.Definitions.DestinyNodeActivationRequirement | If the step has requirements for activation (they almost always do, if nothing else than for the Talent Grid's Progression to have reached a certain level), they will be defined here. |
| `canActivateNextStep` | boolean | There was a time when talent nodes could be activated multiple times, and the effects of subsequent Steps would be compounded on each other, essentially "upgrading" the node. We have moved away from this, but theoretically the capability still exists. I continue to return this in case it is used in the future: if true and this step is the current step in the node, you are allowed to activate the node a second time to receive the benefits of the next step in the node, which will then become the active step. |
| `nextStepIndex` | int32 | The stepIndex of the next step in the talent node, or -1 if this is the last step or if the next step to be chosen is random. This doesn't really matter anymore unless canActivateNextStep begins to be used again. |
| `isNextStepRandom` | boolean | If true, the next step to be chosen is random, and if you're allowed to activate the next step. (if canActivateNextStep = true) |
| `perkHashes` | array&lt;uint32&gt; → DestinySandboxPerkDefinition | The list of hash identifiers for Perks (DestinySandboxPerkDefinition) that are applied when this step is active. Perks provide a variety of benefits and modifications - examine DestinySandboxPerkDefinition to learn more. |
| `startProgressionBarAtProgress` | int32 | When the Talent Grid's progression reaches this value, the circular "progress bar" that surrounds the talent node should be shown. This also indicates the lower bound of said progress bar, with the upper bound being the progress required to reach activationRequirement.gridLevel. (at some point I should precalculate the upper bound and put it in the definition to save people time) |
| `statHashes` | array&lt;uint32&gt; → DestinyStatDefinition | When the step provides stat benefits on the item or character, this is the list of hash identifiers for stats (DestinyStatDefinition) that are provided. |
| `affectsQuality` | boolean | If this is true, the step affects the item's Quality in some way. See DestinyInventoryItemDefinition for more information about the meaning of Quality. I already made a joke about Zen and the Art of Motorcycle Maintenance elsewhere in the documentation, so I will avoid doing it again. Oops too late |
| `stepGroups` | Destiny.Definitions.DestinyTalentNodeStepGroups | In Destiny 1, the Armory's Perk Filtering was driven by a concept of TalentNodeStepGroups: categorizations of talent nodes based on their functionality. While the Armory isn't a BNet-facing thing for now, and the new Armory will need to account for Sockets rather than Talent Nodes, this categorization capability feels useful enough to still keep around. |
| `affectsLevel` | boolean | If true, this step can affect the level of the item. See DestinyInventoryItemDefintion for more information about item levels and their effect on stats. |
| `socketReplacements` | array&lt;Destiny.Definitions.DestinyNodeSocketReplaceResponse&gt; | If this step is activated, this will be a list of information used to replace socket items with new Plugs. See DestinyInventoryItemDefinition for more information about sockets and plugs. |

#### Destiny.Definitions.DestinyTalentNodeStepGroups

**Type:** object

These properties are an attempt to categorize talent node steps by certain common properties. See the related enumerations for the type of properties being categorized.

| Property | Type | Description |
| --- | --- | --- |
| `weaponPerformance` | int32 | — |
| `impactEffects` | int32 | — |
| `guardianAttributes` | int32 | — |
| `lightAbilities` | int32 | — |
| `damageTypes` | int32 | — |

#### Destiny.Definitions.DestinyTalentNodeStepWeaponPerformancesEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `RateOfFire` | 1 | — |
| `Damage` | 2 | — |
| `Accuracy` | 4 | — |
| `Range` | 8 | — |
| `Zoom` | 16 | — |
| `Recoil` | 32 | — |
| `Ready` | 64 | — |
| `Reload` | 128 | — |
| `HairTrigger` | 256 | — |
| `AmmoAndMagazine` | 512 | — |
| `TrackingAndDetonation` | 1024 | — |
| `ShotgunSpread` | 2048 | — |
| `ChargeTime` | 4096 | — |
| `All` | 8191 | — |

#### Destiny.Definitions.DestinyTalentNodeStepImpactEffectsEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `ArmorPiercing` | 1 | — |
| `Ricochet` | 2 | — |
| `Flinch` | 4 | — |
| `CollateralDamage` | 8 | — |
| `Disorient` | 16 | — |
| `HighlightTarget` | 32 | — |
| `All` | 63 | — |

#### Destiny.Definitions.DestinyTalentNodeStepGuardianAttributesEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Stats` | 1 | — |
| `Shields` | 2 | — |
| `Health` | 4 | — |
| `Revive` | 8 | — |
| `AimUnderFire` | 16 | — |
| `Radar` | 32 | — |
| `Invisibility` | 64 | — |
| `Reputations` | 128 | — |
| `All` | 255 | — |

#### Destiny.Definitions.DestinyTalentNodeStepLightAbilitiesEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Grenades` | 1 | — |
| `Melee` | 2 | — |
| `MovementModes` | 4 | — |
| `Orbs` | 8 | — |
| `SuperEnergy` | 16 | — |
| `SuperMods` | 32 | — |
| `All` | 63 | — |

#### Destiny.Definitions.DestinyTalentNodeStepDamageTypesEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Kinetic` | 1 | — |
| `Arc` | 2 | — |
| `Solar` | 4 | — |
| `Void` | 8 | — |
| `All` | 15 | — |

#### Destiny.Definitions.DestinyNodeSocketReplaceResponse

**Type:** object

This is a bit of an odd duck. Apparently, if talent nodes steps have this data, the game will go through on step activation and alter the first Socket it finds on the item that has a type matching the given socket type, inserting the indicated plug item.

| Property | Type | Description |
| --- | --- | --- |
| `socketTypeHash` | uint32 → DestinySocketTypeDefinition | The hash identifier of the socket type to find amidst the item's sockets (the item to which this talent grid is attached). See DestinyInventoryItemDefinition.sockets.socketEntries to find the socket type of sockets on the item in question. |
| `plugItemHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier of the plug item that will be inserted into the socket found. |

#### Destiny.Definitions.DestinyTalentNodeExclusiveSetDefinition

**Type:** object

The list of indexes into the Talent Grid's "nodes" property for nodes in this exclusive set. (See DestinyTalentNodeDefinition.nodeIndex)

| Property | Type | Description |
| --- | --- | --- |
| `nodeIndexes` | array&lt;int32&gt; | The list of node indexes for the exclusive set. Historically, these were indexes. I would have liked to replace this with nodeHashes for consistency, but it's way too late for that. (9:09 PM, he's right!) |

#### Destiny.Definitions.DestinyTalentExclusiveGroup

**Type:** object

As of Destiny 2, nodes can exist as part of "Exclusive Groups". These differ from exclusive sets in that, within the group, many nodes can be activated. But the act of activating any node in the group will cause "opposing" nodes (nodes in groups that are not allowed to be activated at the same time as this group) to deactivate.

| Property | Type | Description |
| --- | --- | --- |
| `groupHash` | uint32 | The identifier for this exclusive group. Only guaranteed unique within the talent grid, not globally. |
| `loreHash` | uint32 → DestinyLoreDefinition? | If this group has an associated piece of lore to show next to it, this will be the identifier for that DestinyLoreDefinition. |
| `nodeHashes` | array&lt;uint32&gt; | A quick reference of the talent nodes that are part of this group, by their Talent Node hashes. (See DestinyTalentNodeDefinition.nodeHash) |
| `opposingGroupHashes` | array&lt;uint32&gt; | A quick reference of Groups whose nodes will be deactivated if any node in this group is activated. |
| `opposingNodeHashes` | array&lt;uint32&gt; | A quick reference of Nodes that will be deactivated if any node in this group is activated, by their Talent Node hashes. (See DestinyTalentNodeDefinition.nodeHash) |

#### Destiny.Definitions.DestinyTalentNodeCategory

**Type:** object

An artificial construct provided by Bungie.Net, where we attempt to group talent nodes by functionality. This is a single set of references to Talent Nodes that share a common trait or purpose.

| Property | Type | Description |
| --- | --- | --- |
| `identifier` | string | Mostly just for debug purposes, but if you find it useful you can have it. This is BNet's manually created identifier for this category. |
| `isLoreDriven` | boolean | If true, we found the localized content in a related DestinyLoreDefinition instead of local BNet localization files. This is mostly for ease of my own future investigations. |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | Will contain at least the "name", which will be the title of the category. We will likely not have description and an icon yet, but I'm going to keep my options open. |
| `nodeHashes` | array&lt;uint32&gt; | The set of all hash identifiers for Talent Nodes (DestinyTalentNodeDefinition) in this Talent Grid that are part of this Category. |

#### Destiny.Definitions.DestinyItemPerkEntryDefinition

**Type:** object

An intrinsic perk on an item, and the requirements for it to be activated.

| Property | Type | Description |
| --- | --- | --- |
| `requirementDisplayString` | string | If this perk is not active, this is the string to show for why it's not providing its benefits. |
| `perkHash` | uint32 → DestinySandboxPerkDefinition | A hash identifier for the DestinySandboxPerkDefinition being provided on the item. |
| `perkVisibility` | int32 | Indicates whether this perk should be shown, or if it should be shown disabled. |

#### Destiny.ItemPerkVisibilityEnumeration

**Enum** (`int32`)

Indicates how a perk should be shown, or if it should be, in the game UI. Maybe useful for those of you trying to filter out internal-use-only perks (or for those of you trying to figure out what they do!)

| Value | # | Description |
| --- | --- | --- |
| `Visible` | 0 | — |
| `Disabled` | 1 | — |
| `Hidden` | 2 | — |

#### Destiny.Definitions.Animations.DestinyAnimationReference

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `animName` | string | — |
| `animIdentifier` | string | — |
| `path` | string | — |

#### Destiny.SpecialItemTypeEnumeration

**Enum** (`int32`)

As you run into items that need to be classified for Milestone purposes in ways that we cannot infer via direct data, add a new classification here and use a string constant to represent it in the local item config file. NOTE: This is not all of the item types available, and some of these are holdovers from Destiny 1 that may or may not still exist.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `SpecialCurrency` | 1 | — |
| `Armor` | 8 | — |
| `Weapon` | 9 | — |
| `Engram` | 23 | — |
| `Consumable` | 24 | — |
| `ExchangeMaterial` | 25 | — |
| `MissionReward` | 27 | — |
| `Currency` | 29 | — |

#### Destiny.DestinyItemTypeEnumeration

**Enum** (`int32`)

An enumeration that indicates the high-level "type" of the item, attempting to iron out the context specific differences for specific instances of an entity. For instance, though a weapon may be of various weapon "Types", in DestinyItemType they are all classified as "Weapon". This allows for better filtering on a higher level of abstraction for the concept of types. This enum is provided for historical compatibility with Destiny 1, but an ideal alternative is to use DestinyItemCategoryDefinitions and the DestinyItemDefinition.itemCategories property instead. Item Categories allow for arbitrary hierarchies of specificity, and for items to belong to multiple categories across multiple hierarchies simultaneously. For this enum, we pick a single type as a "best guess" fit. NOTE: This is not all of the item types available, and some of these are holdovers from Destiny 1 that may or may not still exist. I keep updating these because they're so damn convenient. I guess I shouldn't fight it.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Currency` | 1 | — |
| `Armor` | 2 | — |
| `Weapon` | 3 | — |
| `Message` | 7 | — |
| `Engram` | 8 | — |
| `Consumable` | 9 | — |
| `ExchangeMaterial` | 10 | — |
| `MissionReward` | 11 | — |
| `QuestStep` | 12 | — |
| `QuestStepComplete` | 13 | — |
| `Emblem` | 14 | — |
| `Quest` | 15 | — |
| `Subclass` | 16 | — |
| `ClanBanner` | 17 | — |
| `Aura` | 18 | — |
| `Mod` | 19 | — |
| `Dummy` | 20 | — |
| `Ship` | 21 | — |
| `Vehicle` | 22 | — |
| `Emote` | 23 | — |
| `Ghost` | 24 | — |
| `Package` | 25 | — |
| `Bounty` | 26 | — |
| `Wrapper` | 27 | — |
| `SeasonalArtifact` | 28 | — |
| `Finisher` | 29 | — |
| `Pattern` | 30 | — |

#### Destiny.DestinyBreakerTypeEnumeration

**Enum** (`int32`)

A plug can optionally have a "Breaker Type": a special ability that can affect units in unique ways. Activating this plug can grant one of these types.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `ShieldPiercing` | 1 | — |
| `Disruption` | 2 | — |
| `Stagger` | 3 | — |

#### Destiny.Definitions.DestinyItemCategoryDefinition

**Object** · *(Manifest definition, table `ItemCategories`)*

In an attempt to categorize items by type, usage, and other interesting properties, we created DestinyItemCategoryDefinition: information about types that is assembled using a set of heuristics that examine the properties of an item such as what inventory bucket it's in, its item type name, and whether it has or is missing certain blocks of data. This heuristic is imperfect, however. If you find an item miscategorized, let us know on the Bungie API forums! We then populate all of the categories that we think an item belongs to in its DestinyInventoryItemDefinition.itemCategoryHashes property. You can use that to provide your own custom item filtering, sorting, aggregating... go nuts on it! And let us know if you see more categories that you wish would be added!

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `visible` | boolean | If True, this category should be visible in UI. Sometimes we make categories that we don't think are interesting externally. It's up to you if you want to skip on showing them. |
| `deprecated` | boolean | If True, this category has been deprecated: it may have no items left, or there may be only legacy items that remain in it which are no longer relevant to the game. |
| `shortTitle` | string | A shortened version of the title. The reason why we have this is because the Armory in German had titles that were too long to display in our UI, so these were localized abbreviated versions of those categories. The property still exists today, even though the Armory doesn't exist for D2... yet. |
| `itemTypeRegex` | string | The janky regular expression we used against the item type to try and discern whether the item belongs to this category. |
| `grantDestinyBreakerType` | int32 | If the item in question has this category, it also should have this breaker type. |
| `plugCategoryIdentifier` | string | If the item is a plug, this is the identifier we expect to find associated with it if it is in this category. |
| `itemTypeRegexNot` | string | If the item type matches this janky regex, it does *not* belong to this category. |
| `originBucketIdentifier` | string | If the item belongs to this bucket, it does belong to this category. |
| `grantDestinyItemType` | int32 | If an item belongs to this category, it will also receive this item type. This is now how DestinyItemType is populated for items: it used to be an even jankier process, but that's a story that requires more alcohol. |
| `grantDestinySubType` | int32 | If an item belongs to this category, it will also receive this subtype enum value. I know what you're thinking - what if it belongs to multiple categories that provide sub-types? The last one processed wins, as is the case with all of these "grant" enums. Now you can see one reason why we moved away from these enums... but they're so convenient when they work, aren't they? |
| `grantDestinyClass` | int32 | If an item belongs to this category, it will also get this class restriction enum value. See the other "grant"-prefixed properties on this definition for my color commentary. |
| `traitId` | string | The traitId that can be found on items that belong to this category. |
| `groupedCategoryHashes` | array&lt;uint32&gt; → DestinyItemCategoryDefinition | If this category is a "parent" category of other categories, those children will have their hashes listed in rendering order here, and can be looked up using these hashes against DestinyItemCategoryDefinition. In this way, you can build up a visual hierarchy of item categories. That's what we did, and you can do it too. I believe in you. Yes, you, Carl. (I hope someone named Carl reads this someday) |
| `parentCategoryHashes` | array&lt;uint32&gt; | All item category hashes of "parent" categories: categories that contain this as a child through the hierarchy of groupedCategoryHashes. It's a bit redundant, but having this child-centric list speeds up some calculations. |
| `groupCategoryOnly` | boolean | If true, this category is only used for grouping, and should not be evaluated with its own checks. Rather, the item only has this category if it has one of its child categories. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.BreakerTypes.DestinyBreakerTypeDefinition

**Object** · *(Manifest definition, table `BreakerTypes`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `enumValue` | int32 | We have an enumeration for Breaker types for quick reference. This is the current definition's breaker type enum value. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Seasons.DestinySeasonDefinition

**Object** · *(Manifest definition, table `Seasons`)*

Defines a canonical "Season" of Destiny: a range of a few months where the game highlights certain challenges, provides new loot, has new Clan-related rewards and celebrates various seasonal events.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `backgroundImagePath` | string | — |
| `seasonNumber` | int32 | — |
| `startDate` | date-time? | — |
| `endDate` | date-time? | — |
| `seasonPassHash` | uint32 → DestinySeasonPassDefinition? | — |
| `seasonPassList` | array&lt;Destiny.Definitions.Seasons.DestinySeasonPassReference&gt; → DestinySeasonPassDefinition | — |
| `seasonPassProgressionHash` | uint32 → DestinyProgressionDefinition? | — |
| `artifactItemHash` | uint32 → DestinyInventoryItemDefinition? | — |
| `sealPresentationNodeHash` | uint32 → DestinyPresentationNodeDefinition? | — |
| `acts` | array&lt;Destiny.Definitions.Seasons.DestinySeasonActDefinition&gt; | A list of Acts for the Episode |
| `seasonalChallengesPresentationNodeHash` | uint32 → DestinyPresentationNodeDefinition? | — |
| `preview` | Destiny.Definitions.Seasons.DestinySeasonPreviewDefinition | Optional - Defines the promotional text, images, and links to preview this season. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Seasons.DestinySeasonPassReference

**Type:** object

Defines the hash, unlock flag and start time of season passes

| Property | Type | Description |
| --- | --- | --- |
| `seasonPassHash` | uint32 → DestinySeasonPassDefinition | The Season Pass Hash |
| `seasonPassStartDate` | date-time? | The Season Pass Start Date |
| `seasonPassEndDate` | date-time? | The Season Pass End Date |

#### Destiny.Definitions.Seasons.DestinySeasonPassDefinition

**Object** · *(Manifest definition, table `SeasonPasses`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `rewardProgressionHash` | uint32 → DestinyProgressionDefinition | This is the progression definition related to the progression for the initial levels 1-100 that provide item rewards for the Season pass. Further experience after you reach the limit is provided in the "Prestige" progression referred to by prestigeProgressionHash. |
| `prestigeProgressionHash` | uint32 → DestinyProgressionDefinition | I know what you're thinking, but I promise we're not going to duplicate and drown you. Instead, we're giving you sweet, sweet power bonuses. Prestige progression is further progression that you can make on the Season pass after you gain max ranks, that will ultimately increase your power/light level over the theoretical limit. |
| `linkRedirectPath` | string | — |
| `color` | Destiny.Misc.DestinyColor | — |
| `images` | Destiny.Definitions.Seasons.DestinySeasonPassImages | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Seasons.DestinySeasonPassImages

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `iconImagePath` | string | — |
| `themeBackgroundImagePath` | string | — |

#### Destiny.Definitions.Seasons.DestinySeasonActDefinition

**Type:** object

Defines the name, start time and ranks included in an Act of an Episode.

| Property | Type | Description |
| --- | --- | --- |
| `displayName` | string | The name of the Act. |
| `startTime` | date-time | The start time of the Act. |
| `rankCount` | int32 | The number of ranks included in the Act. |

#### Destiny.Definitions.Seasons.DestinySeasonPreviewDefinition

**Type:** object

Defines the promotional text, images, and links to preview this season.

| Property | Type | Description |
| --- | --- | --- |
| `description` | string | A localized description of the season. |
| `linkPath` | string | A relative path to learn more about the season. Web browsers should be automatically redirected to the user's Bungie.net locale. For example: "/SeasonOfTheChosen" will redirect to "/7/en/ Seasons/SeasonOfTheChosen" for English users. |
| `videoLink` | string | An optional link to a localized video, probably YouTube. |
| `images` | array&lt;Destiny.Definitions.Seasons.DestinySeasonPreviewImageDefinition&gt; | A list of images to preview the seasonal content. Should have at least three to show. |

#### Destiny.Definitions.Seasons.DestinySeasonPreviewImageDefinition

**Type:** object

Defines the thumbnail icon, high-res image, and video link for promotional images

| Property | Type | Description |
| --- | --- | --- |
| `thumbnailImage` | string | A thumbnail icon path to preview seasonal content, probably 480x270. |
| `highResImage` | string | An optional path to a high-resolution image, probably 1920x1080. |

#### Destiny.Definitions.DestinyProgressionRewardItemQuantity

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `rewardItemIndex` | int32 | — |
| `rewardedAtProgressionLevel` | int32 | — |
| `acquisitionBehavior` | int32 | — |
| `uiDisplayStyle` | string | — |
| `claimUnlockDisplayStrings` | array&lt;string&gt; | — |
| `socketOverrides` | array&lt;Destiny.Definitions.DestinyProgressionSocketPlugOverride&gt; | — |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier for the item in question. Use it to look up the item's DestinyInventoryItemDefinition. |
| `itemInstanceId` | int64? | If this quantity is referring to a specific instance of an item, this will have the item's instance ID. Normally, this will be null. |
| `quantity` | int32 | The amount of the item needed/available depending on the context of where DestinyItemQuantity is being used. |
| `hasConditionalVisibility` | boolean | Indicates that this item quantity may be conditionally shown or hidden, based on various sources of state. For example: server flags, account state, or character progress. |

#### Destiny.DestinyProgressionRewardItemAcquisitionBehaviorEnumeration

**Enum** (`int32`)

Represents the different kinds of acquisition behavior for progression reward items.

| Value | # | Description |
| --- | --- | --- |
| `Instant` | 0 | — |
| `PlayerClaimRequired` | 1 | — |

#### Destiny.Definitions.DestinyProgressionSocketPlugOverride

**Type:** object

The information for how progression item definitions should override a given socket with custom plug data.

| Property | Type | Description |
| --- | --- | --- |
| `socketTypeHash` | uint32 | — |
| `overrideSingleItemHash` | uint32 → DestinyInventoryItemDefinition? | — |

#### Destiny.Config.DestinyManifest

**Type:** object

DestinyManifest is the external-facing contract for just the properties needed by those calling the Destiny Platform.

| Property | Type | Description |
| --- | --- | --- |
| `version` | string | — |
| `mobileAssetContentPath` | string | — |
| `mobileGearAssetDataBases` | array&lt;Destiny.Config.GearAssetDataBaseDefinition&gt; | — |
| `mobileWorldContentPaths` | Mapping&lt;string, string&gt; | — |
| `jsonWorldContentPaths` | Mapping&lt;string, string&gt; | This points to the generated JSON that contains all the Definitions. Each key is a locale. The value is a path to the aggregated world definitions (warning: large file!) |
| `jsonWorldComponentContentPaths` | Mapping&lt;string, object&gt; | This points to the generated JSON that contains all the Definitions. Each key is a locale. The value is a dictionary, where the key is a definition type by name, and the value is the path to the file for that definition. WARNING: This is unsafe and subject to change - do not depend on data in these files staying around long-term. |
| `mobileClanBannerDatabasePath` | string | — |
| `mobileGearCDN` | Mapping&lt;string, string&gt; | — |
| `iconImagePyramidInfo` | array&lt;Destiny.Config.ImagePyramidEntry&gt; | Information about the "Image Pyramid" for Destiny icons. Where possible, we create smaller versions of Destiny icons. These are found as subfolders under the location of the "original/full size" Destiny images, with the same file name and extension as the original image itself. (this lets us avoid sending largely redundant path info with every entity, at the expense of the smaller versions of the image being less discoverable) |

#### Destiny.Config.GearAssetDataBaseDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `version` | int32 | — |
| `path` | string | — |

#### Destiny.Config.ImagePyramidEntry

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `name` | string | The name of the subfolder where these images are located. |
| `factor` | float | The factor by which the original image size has been reduced. |

#### Destiny.Responses.DestinyLinkedProfilesResponse

**Type:** object

I know what you seek. You seek linked accounts. Found them, you have. This contract returns a minimal amount of data about Destiny Accounts that are linked through your Bungie.Net account. We will not return accounts in this response whose

| Property | Type | Description |
| --- | --- | --- |
| `profiles` | array&lt;Destiny.Responses.DestinyProfileUserInfoCard&gt; | Any Destiny account for whom we could successfully pull characters will be returned here, as the Platform-level summary of user data. (no character data, no Destiny account data other than the Membership ID and Type so you can make further queries) |
| `bnetMembership` | User.UserInfoCard | If the requested membership had a linked Bungie.Net membership ID, this is the basic information about that BNet account. I know, Tetron; I know this is mixing UserServices concerns with DestinyServices concerns. But it's so damn convenient! <https://www.youtube.com/watch?v=X5R-bB-gKVI> |
| `profilesWithErrors` | array&lt;Destiny.Responses.DestinyErrorProfile&gt; | This is brief summary info for profiles that we believe have valid Destiny info, but who failed to return data for some other reason and thus we know that subsequent calls for their info will also fail. |

#### Destiny.Responses.DestinyProfileUserInfoCard

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `dateLastPlayed` | date-time | — |
| `isOverridden` | boolean | If this profile is being overridden/obscured by Cross Save, this will be set to true. We will still return the profile for display purposes where users need to know the info: it is up to any given area of the app/site to determine if this profile should still be shown. |
| `isCrossSavePrimary` | boolean | If true, this account is hooked up as the "Primary" cross save account for one or more platforms. |
| `platformSilver` | Destiny.Components.Inventory.DestinyPlatformSilverComponent | This is the silver available on this Profile across any platforms on which they have purchased silver. This is only available if you are requesting yourself. |
| `unpairedGameVersions` | int32? | If this profile is not in a cross save pairing, this will return the game versions that we believe this profile has access to. For the time being, we will not return this information for any membership that is in a cross save pairing. The gist is that, once the pairing occurs, we do not currently have a consistent way to get that information for the profile's original Platform, and thus gameVersions would be too inconsistent (based on the last platform they happened to play on) for the info to be useful. If we ever can get this data, this field will be deprecated and replaced with data on the DestinyLinkedProfileResponse itself, with game versions per linked Platform. But since we can't get that, we have this as a stop-gap measure for getting the data in the only situation that we currently need it. |
| `supplementalDisplayName` | string | A platform specific additional display name - ex: psn Real Name, bnet Unique Name, etc. |
| `iconPath` | string | URL the Icon if available. |
| `crossSaveOverride` | int32 | If there is a cross save override in effect, this value will tell you the type that is overridding this one. |
| `applicableMembershipTypes` | array&lt;int32&gt; | The list of Membership Types indicating the platforms on which this Membership can be used. Not in Cross Save = its original membership type. Cross Save Primary = Any membership types it is overridding, and its original membership type Cross Save Overridden = Empty list |
| `isPublic` | boolean | If True, this is a public user membership. |
| `membershipType` | int32 | Type of the membership. Not necessarily the native type. |
| `membershipId` | int64 | Membership ID as they user is known in the Accounts service |
| `displayName` | string | Display Name the player has chosen for themselves. The display name is optional when the data type is used as input to a platform API. |
| `bungieGlobalDisplayName` | string | The bungie global display name, if set. |
| `bungieGlobalDisplayNameCode` | int16? | The bungie global display name code, if set. |

#### Destiny.Components.Inventory.DestinyPlatformSilverComponentDepends on Component "PlatformSilver"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `platformSilver` | Mapping&lt;int32, Destiny.Entities.Items.DestinyItemComponent&gt; | If a Profile is played on multiple platforms, this is the silver they have for each platform, keyed by Membership Type. |

#### Destiny.Entities.Items.DestinyItemComponent

**Type:** object

The base item component, filled with properties that are generally useful to know in any item request or that don't feel worthwhile to put in their own component.

| Property | Type | Description |
| --- | --- | --- |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | The identifier for the item's definition, which is where most of the useful static information for the item can be found. |
| `itemInstanceId` | int64? | If the item is instanced, it will have an instance ID. Lack of an instance ID implies that the item has no distinct local qualities aside from stack size. |
| `quantity` | int32 | The quantity of the item in this stack. Note that Instanced items cannot stack. If an instanced item, this value will always be 1 (as the stack has exactly one item in it) |
| `bindStatus` | int32 | If the item is bound to a location, it will be specified in this enum. |
| `location` | int32 | An easy reference for where the item is located. Redundant if you got the item from an Inventory, but useful when making detail calls on specific items. |
| `bucketHash` | uint32 → DestinyInventoryBucketDefinition | The hash identifier for the specific inventory bucket in which the item is located. |
| `transferStatus` | int32 | If there is a known error state that would cause this item to not be transferable, this Flags enum will indicate all of those error states. Otherwise, it will be 0 (CanTransfer). |
| `lockable` | boolean | If the item can be locked, this will indicate that state. |
| `state` | int32 | A flags enumeration indicating the transient/custom states of the item that affect how it is rendered: whether it's tracked or locked for example, or whether it has a masterwork plug inserted. |
| `overrideStyleItemHash` | uint32 → DestinyInventoryItemDefinition? | If populated, this is the hash of the item whose icon (and other secondary styles, but *not* the human readable strings) should override whatever icons/styles are on the item being sold. If you don't do this, certain items whose styles are being overridden by socketed items - such as the "Recycle Shader" item - would show whatever their default icon/style is, and it wouldn't be pretty or look accurate. |
| `expirationDate` | date-time? | If the item can expire, this is the date at which it will/did expire. |
| `isWrapper` | boolean | If this is true, the object is actually a "wrapper" of the object it's representing. This means that it's not the actual item itself, but rather an item that must be "opened" in game before you have and can use the item. Wrappers are an evolution of "bundles", which give an easy way to let you preview the contents of what you purchased while still letting you get a refund before you "open" it. |
| `tooltipNotificationIndexes` | array&lt;int32&gt; | If this is populated, it is a list of indexes into DestinyInventoryItemDefinition.tooltipNotifications for any special tooltip messages that need to be shown for this item. |
| `metricHash` | uint32 → DestinyMetricDefinition? | The identifier for the currently-selected metric definition, to be displayed on the emblem nameplate. |
| `metricObjective` | Destiny.Quests.DestinyObjectiveProgress | The objective progress for the currently-selected metric definition, to be displayed on the emblem nameplate. |
| `versionNumber` | int32? | The version of this item, used to index into the versions list in the item definition quality block. |
| `itemValueVisibility` | array&lt;boolean&gt; | If available, a list that describes which item values (rewards) should be shown (true) or hidden (false). |

#### Destiny.ItemBindStatusEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `NotBound` | 0 | — |
| `BoundToCharacter` | 1 | — |
| `BoundToAccount` | 2 | — |
| `BoundToGuild` | 3 | — |

#### Destiny.TransferStatusesEnumeration

**Enum** (`int32`)

Whether you can transfer an item, and why not if you can't.

| Value | # | Description |
| --- | --- | --- |
| `CanTransfer` | 0 | The item can be transferred. |
| `ItemIsEquipped` | 1 | You can't transfer the item because it is equipped on a character. |
| `NotTransferrable` | 2 | The item is defined as not transferrable in its DestinyInventoryItemDefinition.nonTransferrable property. |
| `NoRoomInDestination` | 4 | You could transfer the item, but the place you're trying to put it has run out of room! Check your remaining Vault and/or character space. |

#### Destiny.Quests.DestinyObjectiveProgress

**Type:** object

Returns data about a character's status with a given Objective. Combine with DestinyObjectiveDefinition static data for display purposes.

| Property | Type | Description |
| --- | --- | --- |
| `objectiveHash` | uint32 → DestinyObjectiveDefinition | The unique identifier of the Objective being referred to. Use to look up the DestinyObjectiveDefinition in static data. |
| `destinationHash` | uint32 → DestinyDestinationDefinition? | If the Objective has a Destination associated with it, this is the unique identifier of the Destination being referred to. Use to look up the DestinyDestinationDefinition in static data. This will give localized data about *where* in the universe the objective should be achieved. |
| `activityHash` | uint32 → DestinyActivityDefinition? | If the Objective has an Activity associated with it, this is the unique identifier of the Activity being referred to. Use to look up the DestinyActivityDefinition in static data. This will give localized data about *what* you should be playing for the objective to be achieved. |
| `progress` | int32? | If progress has been made, and the progress can be measured numerically, this will be the value of that progress. You can compare it to the DestinyObjectiveDefinition.completionValue property for current vs. upper bounds, and use DestinyObjectiveDefinition.inProgressValueStyle or completedValueStyle to determine how this should be rendered. Note that progress, in Destiny 2, need not be a literal numeric progression. It could be one of a number of possible values, even a Timestamp. Always examine DestinyObjectiveDefinition.inProgressValueStyle or completedValueStyle before rendering progress. |
| `completionValue` | int32 | As of Forsaken, objectives' completion value is determined dynamically at runtime. This value represents the threshold of progress you need to surpass in order for this objective to be considered "complete". If you were using objective data, switch from using the DestinyObjectiveDefinition's "completionValue" to this value. |
| `complete` | boolean | Whether or not the Objective is completed. |
| `visible` | boolean | If this is true, the objective is visible in-game. Otherwise, it's not yet visible to the player. Up to you if you want to honor this property. |

#### Destiny.DestinyGameVersionsEnumeration

**Enum** (`int32`)

A flags enumeration/bitmask indicating the versions of the game that a given user has purchased.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Destiny2` | 1 | — |
| `DLC1` | 2 | — |
| `DLC2` | 4 | — |
| `Forsaken` | 8 | — |
| `YearTwoAnnualPass` | 16 | — |
| `Shadowkeep` | 32 | — |
| `BeyondLight` | 64 | — |
| `Anniversary30th` | 128 | — |
| `TheWitchQueen` | 256 | — |
| `Lightfall` | 512 | — |
| `TheFinalShape` | 1024 | — |
| `EdgeOfFate` | 2048 | — |
| `Renegades` | 4096 | — |

#### Destiny.Responses.DestinyErrorProfile

**Type:** object

If a Destiny Profile can't be returned, but we're pretty certain it's a valid Destiny account, this will contain as much info as we can get about the profile for your use. Assume that the most you'll get is the Error Code, the Membership Type and the Membership ID.

| Property | Type | Description |
| --- | --- | --- |
| `errorCode` | int32 | The error that we encountered. You should be able to look up localized text to show to the user for these failures. |
| `infoCard` | User.UserInfoCard | Basic info about the account that failed. Don't expect anything other than membership ID, Membership Type, and displayName to be populated. |

#### Destiny.DestinyComponentTypeEnumeration

**Enum** (`int32`)

Represents the possible components that can be returned from Destiny "Get" calls such as GetProfile, GetCharacter, GetVendor etc... When making one of these requests, you will pass one or more of these components as a comma separated list in the "?components=" querystring parameter. For instance, if you want baseline Profile data, Character Data, and character progressions, you would pass "? components=Profiles,Characters,CharacterProgressions" You may use either the numerical or string values.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Profiles` | 100 | Profiles is the most basic component, only relevant when calling GetProfile. This returns basic information about the profile, which is almost nothing: a list of characterIds, some information about the last time you logged in, and that most sobering statistic: how long you've played. |
| `VendorReceipts` | 101 | Only applicable for GetProfile, this will return information about receipts for refundable vendor items. |
| `ProfileInventories` | 102 | Asking for this will get you the profile-level inventories, such as your Vault buckets (yeah, the Vault is really inventory buckets located on your Profile) |
| `ProfileCurrencies` | 103 | This will get you a summary of items on your Profile that we consider to be "currencies", such as Glimmer. I mean, if there's Glimmer in Destiny 2. I didn't say there was Glimmer. |
| `ProfileProgression` | 104 | This will get you any progression-related information that exists on a Profile-wide level, across all characters. |
| `PlatformSilver` | 105 | This will get you information about the silver that this profile has on every platform on which it plays. You may only request this component for the logged in user's Profile, and will not recieve it if you request it for another Profile. |
| `Characters` | 200 | This will get you summary info about each of the characters in the profile. |
| `CharacterInventories` | 201 | This will get you information about any non-equipped items on the character or character(s) in question, if you're allowed to see it. You have to either be authenticated as that user, or that user must allow anonymous viewing of their non-equipped items in Bungie.Net settings to actually get results. |
| `CharacterProgressions` | 202 | This will get you information about the progression (faction, experience, etc... "levels") relevant to each character, if you are the currently authenticated user or the user has elected to allow anonymous viewing of its progression info. |
| `CharacterRenderData` | 203 | This will get you just enough information to be able to render the character in 3D if you have written a 3D rendering library for Destiny Characters, or "borrowed" ours. It's okay, I won't tell anyone if you're using it. I'm no snitch. (actually, we don't care if you use it - go to town) |
| `CharacterActivities` | 204 | This will return info about activities that a user can see and gating on it, if you are the currently authenticated user or the user has elected to allow anonymous viewing of its progression info. Note that the data returned by this can be unfortunately problematic and relatively unreliable in some cases. We'll eventually work on making it more consistently reliable. |
| `CharacterEquipment` | 205 | This will return info about the equipped items on the character(s). Everyone can see this. |
| `CharacterLoadouts` | 206 | This will return info about the loadouts of the character(s). |
| `ItemInstances` | 300 | This will return basic info about instanced items - whether they can be equipped, their tracked status, and some info commonly needed in many places (current damage type, primary stat value, etc) |
| `ItemObjectives` | 301 | Items can have Objectives (DestinyObjectiveDefinition) bound to them. If they do, this will return info for items that have such bound objectives. |
| `ItemPerks` | 302 | Items can have perks (DestinySandboxPerkDefinition). If they do, this will return info for what perks are active on items. |
| `ItemRenderData` | 303 | If you just want to render the weapon, this is just enough info to do that rendering. |
| `ItemStats` | 304 | Items can have stats, like rate of fire. Asking for this component will return requested item's stats if they have stats. |
| `ItemSockets` | 305 | Items can have sockets, where plugs can be inserted. Asking for this component will return all info relevant to the sockets on items that have them. |
| `ItemTalentGrids` | 306 | Items can have talent grids, though that matters a lot less frequently than it used to. Asking for this component will return all relevant info about activated Nodes and Steps on this talent grid, like the good ol' days. |
| `ItemCommonData` | 307 | Items that *aren't* instanced still have important information you need to know: how much of it you have, the itemHash so you can look up their DestinyInventoryItemDefinition, whether they're locked, etc... Both instanced and non-instanced items will have these properties. You will get this automatically with Inventory components - you only need to pass this when calling GetItem on a specific item. |
| `ItemPlugStates` | 308 | Items that are "Plugs" can be inserted into sockets. This returns statuses about those plugs and why they can/can't be inserted. I hear you giggling, there's nothing funny about inserting plugs. Get your head out of the gutter and pay attention! |
| `ItemPlugObjectives` | 309 | Sometimes, plugs have objectives on them. This data can get really large, so we split it into its own component. Please, don't grab it unless you need it. |
| `ItemReusablePlugs` | 310 | Sometimes, designers create thousands of reusable plugs and suddenly your response sizes are almost 3MB, and something has to give. Reusable Plugs were split off as their own component, away from ItemSockets, as a result of the Plug changes in Shadowkeep that made plug data infeasibly large for the most common use cases. Request this component if and only if you need to know what plugs *could* be inserted into a socket, and need to know it before "drilling" into the details of an item in your application (for instance, if you're doing some sort of interesting sorting or aggregation based on available plugs. When you get this, you will also need to combine it with "Plug Sets" data if you want a full picture of all of the available plugs: this component will only return plugs that have state data that is per- item. See Plug Sets for available plugs that have Character, Profile, or no state-specific restrictions. |
| `Vendors` | 400 | When obtaining vendor information, this will return summary information about the Vendor or Vendors being returned. |
| `VendorCategories` | 401 | When obtaining vendor information, this will return information about the categories of items provided by the Vendor. |
| `VendorSales` | 402 | When obtaining vendor information, this will return the information about items being sold by the Vendor. |
| `Kiosks` | 500 | Asking for this component will return you the account's Kiosk statuses: that is, what items have been filled out/acquired. But only if you are the currently authenticated user or the user has elected to allow anonymous viewing of its progression info. |
| `CurrencyLookups` | 600 | A "shortcut" component that will give you all of the item hashes/quantities of items that the requested character can use to determine if an action (purchasing, socket insertion) has the required currency. (recall that all currencies are just items, and that some vendor purchases require items that you might not traditionally consider to be a "currency", like plugs/mods!) |
| `PresentationNodes` | 700 | Returns summary status information about all "Presentation Nodes". See DestinyPresentationNodeDefinition for more details, but the gist is that these are entities used by the game UI to bucket Collectibles and Records into a hierarchy of categories. You may ask for and use this data if you want to perform similar bucketing in your own UI: or you can skip it and roll your own. |
| `Collectibles` | 800 | Returns summary status information about all "Collectibles". These are records of what items you've discovered while playing Destiny, and some other basic information. For detailed information, you will have to call a separate endpoint devoted to the purpose. |
| `Records` | 900 | Returns summary status information about all "Records" (also known in the game as "Triumphs". I know, it's confusing because there's also "Moments of Triumph" that will themselves be represented as "Triumphs.") |
| `Transitory` | 1000 | Returns information that Bungie considers to be "Transitory": data that may change too frequently or come from a non-authoritative source such that we don't consider the data to be fully trustworthy, but that might prove useful for some limited use cases. We can provide no guarantee of timeliness nor consistency for this data: buyer beware with the Transitory component. |
| `Metrics` | 1100 | Returns summary status information about all "Metrics" (also known in the game as "Stat Trackers"). |
| `StringVariables` | 1200 | Returns a mapping of localized string variable hashes to values, on a per-account or per- character basis. |
| `Craftables` | 1300 | Returns summary status information about all "Craftables" aka crafting recipe items. |
| `SocialCommendations` | 1400 | Returns score values for all commendations and commendation nodes. |

#### Destiny.Responses.DestinyProfileResponse

**Type:** object

The response for GetDestinyProfile, with components for character and item-level data.

| Property | Type | Description |
| --- | --- | --- |
| `responseMintedTimestamp` | date-time | Records the timestamp of when most components were last generated from the world server source. Unless the component type is specified in the documentation for secondaryComponentsMintedTimestamp, this value is sufficient to do data freshness. |
| `secondaryComponentsMintedTimestamp` | date-time | Some secondary components are not tracked in the primary response timestamp and have their timestamp tracked here. If your component is any of the following, this field is where you will find your timestamp value: PresentationNodes, Records, Collectibles, Metrics, StringVariables, Craftables, Transitory All other component types may use the primary timestamp property. |
| `vendorReceipts` | SingleComponentResponseOfDestinyVendorReceiptsComponent | Recent, refundable purchases you have made from vendors. When will you use it? Couldn't say... COMPONENT TYPE: VendorReceipts |
| `profileInventory` | SingleComponentResponseOfDestinyInventoryComponent | The profile-level inventory of the Destiny Profile. COMPONENT TYPE: ProfileInventories |
| `profileCurrencies` | SingleComponentResponseOfDestinyInventoryComponent | The profile-level currencies owned by the Destiny Profile. COMPONENT TYPE: ProfileCurrencies |
| `profile` | SingleComponentResponseOfDestinyProfileComponent | The basic information about the Destiny Profile (formerly "Account"). COMPONENT TYPE: Profiles |
| `platformSilver` | SingleComponentResponseOfDestinyPlatformSilverComponent | Silver quantities for any platform on which this Profile plays destiny. COMPONENT TYPE: PlatformSilver |
| `profileKiosks` | SingleComponentResponseOfDestinyKiosksComponent | Items available from Kiosks that are available Profile-wide (i.e. across all characters) This component returns information about what Kiosk items are available to you on a *Profile* level. It is theoretically possible for Kiosks to have items gated by specific Character as well. If you ever have those, you will find them on the characterKiosks property. COMPONENT TYPE: Kiosks |
| `profilePlugSets` | SingleComponentResponseOfDestinyPlugSetsComponent | When sockets refer to reusable Plug Sets (see DestinyPlugSetDefinition for more info), this is the set of plugs and their states that are profile-scoped. This comes back with ItemSockets, as it is needed for a complete picture of the sockets on requested items. COMPONENT TYPE: ItemSockets |
| `profileProgression` | SingleComponentResponseOfDestinyProfileProgressionComponent | When we have progression information - such as Checklists - that may apply profile-wide, it will be returned here rather than in the per-character progression data. COMPONENT TYPE: ProfileProgression |
| `profilePresentationNodes` | SingleComponentResponseOfDestinyPresentationNodesComponent | COMPONENT TYPE: PresentationNodes |
| `profileRecords` | SingleComponentResponseOfDestinyProfileRecordsComponent | COMPONENT TYPE: Records |
| `profileCollectibles` | SingleComponentResponseOfDestinyProfileCollectiblesComponent | COMPONENT TYPE: Collectibles |
| `profileTransitoryData` | SingleComponentResponseOfDestinyProfileTransitoryComponent | COMPONENT TYPE: Transitory |
| `metrics` | SingleComponentResponseOfDestinyMetricsComponent | COMPONENT TYPE: Metrics |
| `profileStringVariables` | SingleComponentResponseOfDestinyStringVariablesComponent | COMPONENT TYPE: StringVariables |
| `profileCommendations` | SingleComponentResponseOfDestinySocialCommendationsComponent | COMPONENT TYPE: SocialCommendations |
| `characters` | DictionaryComponentResponseOfint64AndDestinyCharacterComponent | Basic information about each character, keyed by the CharacterId. COMPONENT TYPE: Characters |
| `characterInventories` | DictionaryComponentResponseOfint64AndDestinyInventoryComponent | The character-level non-equipped inventory items, keyed by the Character's Id. COMPONENT TYPE: CharacterInventories |
| `characterLoadouts` | DictionaryComponentResponseOfint64AndDestinyLoadoutsComponent | The character loadouts, keyed by the Character's Id. COMPONENT TYPE: CharacterLoadouts |
| `characterProgressions` | DictionaryComponentResponseOfint64AndDestinyCharacterProgressionComponent | Character-level progression data, keyed by the Character's Id. COMPONENT TYPE: CharacterProgressions |
| `characterRenderData` | DictionaryComponentResponseOfint64AndDestinyCharacterRenderComponent | Character rendering data - a minimal set of info needed to render a character in 3D - keyed by the Character's Id. COMPONENT TYPE: CharacterRenderData |
| `characterActivities` | DictionaryComponentResponseOfint64AndDestinyCharacterActivitiesComponent | Character activity data - the activities available to this character and its status, keyed by the Character's Id. COMPONENT TYPE: CharacterActivities |
| `characterEquipment` | DictionaryComponentResponseOfint64AndDestinyInventoryComponent | The character's equipped items, keyed by the Character's Id. COMPONENT TYPE: CharacterEquipment |
| `characterKiosks` | DictionaryComponentResponseOfint64AndDestinyKiosksComponent | Items available from Kiosks that are available to a specific character as opposed to the account as a whole. It must be combined with data from the profileKiosks property to get a full picture of the character's available items to check out of a kiosk. This component returns information about what Kiosk items are available to you on a *Character* level. Usually, kiosk items will be earned for the entire Profile (all characters) at once. To find those, look in the profileKiosks property. COMPONENT TYPE: Kiosks |
| `characterPlugSets` | DictionaryComponentResponseOfint64AndDestinyPlugSetsComponent | When sockets refer to reusable Plug Sets (see DestinyPlugSetDefinition for more info), this is the set of plugs and their states, per character, that are character-scoped. This comes back with ItemSockets, as it is needed for a complete picture of the sockets on requested items. COMPONENT TYPE: ItemSockets |
| `characterUninstancedItemComponents` | Mapping&lt;int64, DestinyBaseItemComponentSetOfuint32&gt; | Do you ever get the feeling that a system was designed *too* flexibly? That it can be used in so many different ways that you end up being unable to provide an easy to use abstraction for the mess that's happening under the surface? Let's talk about character-specific data that might be related to items without instances. These two statements are totally unrelated, I promise. At some point during D2, it was decided that items - such as Bounties - could be given to characters and *not* have instance data, but that *could* display and even use relevant state information on your account and character. Up to now, any item that had meaningful dependencies on character or account state had to be instanced, and thus "itemComponents" was all that you needed: it was keyed by item's instance IDs and provided the stateful information you needed inside. Unfortunately, we don't live in such a magical world anymore. This is information held on a per- character basis about non-instanced items that the characters have in their inventory - or that reference character-specific state information even if it's in Account-level inventory - and the values related to that item's state in relation to the given character. To give a concrete example, look at a Moments of Triumph bounty. They exist in a character's inventory, and show/care about a character's progression toward completing the bounty. But the bounty itself is a non-instanced item, like a mod or a currency. This returns that data for the characters who have the bounty in their inventory. I'm not crying, you're crying Okay we're both crying but it's going to be okay I promise Actually I shouldn't promise that, I don't know if it's going to be okay |
| `characterPresentationNodes` | DictionaryComponentResponseOfint64AndDestinyPresentationNodesComponent | COMPONENT TYPE: PresentationNodes |
| `characterRecords` | DictionaryComponentResponseOfint64AndDestinyCharacterRecordsComponent | COMPONENT TYPE: Records |
| `characterCollectibles` | DictionaryComponentResponseOfint64AndDestinyCollectiblesComponent | COMPONENT TYPE: Collectibles |
| `characterStringVariables` | DictionaryComponentResponseOfint64AndDestinyStringVariablesComponent | COMPONENT TYPE: StringVariables |
| `characterCraftables` | DictionaryComponentResponseOfint64AndDestinyCraftablesComponent | COMPONENT TYPE: Craftables |
| `itemComponents` | DestinyItemComponentSetOfint64 | Information about instanced items across all returned characters, keyed by the item's instance ID. COMPONENT TYPE: [See inside the DestinyItemComponentSet contract for component types.] |
| `characterCurrencyLookups` | DictionaryComponentResponseOfint64AndDestinyCurrenciesComponent | A "lookup" convenience component that can be used to quickly check if the character has access to items that can be used for purchasing. COMPONENT TYPE: CurrencyLookups |

#### Destiny.Entities.Profiles.DestinyVendorReceiptsComponentDepends on Component "VendorReceipts"

**Type:** object

For now, this isn't used for much: it's a record of the recent refundable purchases that the user has made. In the future, it could be used for providing refunds/buyback via the API. Wouldn't that be fun?

| Property | Type | Description |
| --- | --- | --- |
| `receipts` | array&lt;Destiny.Vendors.DestinyVendorReceipt&gt; | The receipts for refundable purchases made at a vendor. |

#### Destiny.Vendors.DestinyVendorReceipt

**Type:** object

If a character purchased an item that is refundable, a Vendor Receipt will be created on the user's Destiny Profile. These expire after a configurable period of time, but until then can be used to get refunds on items. BNet does not provide the ability to refund a purchase *yet*, but you know.

| Property | Type | Description |
| --- | --- | --- |
| `currencyPaid` | array&lt;Destiny.DestinyItemQuantity&gt; | The amount paid for the item, in terms of items that were consumed in the purchase and their quantity. |
| `itemReceived` | Destiny.DestinyItemQuantity | The item that was received, and its quantity. |
| `licenseUnlockHash` | uint32 | The unlock flag used to determine whether you still have the purchased item. |
| `purchasedByCharacterId` | int64 | The ID of the character who made the purchase. |
| `refundPolicy` | int32 | Whether you can get a refund, and what happens in order for the refund to be received. See the DestinyVendorItemRefundPolicy enum for details. |
| `sequenceNumber` | int32 | The identifier of this receipt. |
| `timeToExpiration` | int64 | The seconds since epoch at which this receipt is rendered invalid. |
| `expiresOn` | date-time | The date at which this receipt is rendered invalid. |

#### Destiny.Entities.Inventory.DestinyInventoryComponent

**Type:** object

A list of minimal information for items in an inventory: be it a character's inventory, or a Profile's inventory. (Note that the Vault is a collection of inventory buckets in the Profile's inventory) Inventory Items returned here are in a flat list, but importantly they have a bucketHash property that indicates the specific inventory bucket that is holding them. These buckets constitute things like the separate sections of the Vault, the user's inventory slots, etc. See DestinyInventoryBucketDefinition for more info.

| Property | Type | Description |
| --- | --- | --- |
| `items` | array&lt;Destiny.Entities.Items.DestinyItemComponent&gt; | The items in this inventory. If you care to bucket them, use the item's bucketHash property to group them. |

#### Destiny.Entities.Profiles.DestinyProfileComponentDepends on Component "Profiles"

**Type:** object

The most essential summary information about a Profile (in Destiny 1, we called these "Accounts").

| Property | Type | Description |
| --- | --- | --- |
| `userInfo` | User.UserInfoCard | If you need to render the Profile (their platform name, icon, etc...) somewhere, this property contains that information. |
| `dateLastPlayed` | date-time | The last time the user played with any character on this Profile. |
| `versionsOwned` | int32 | If you want to know what expansions they own, this will contain that data. IMPORTANT: This field may not return the data you're interested in for Cross-Saved users. It returns the last ownership data we saw for this account - which is to say, what they've purchased on the platform on which they last played, which now could be a different platform. If you don't care about per-platform ownership and only care about whatever platform it seems they are playing on most recently, then this should be "good enough." Otherwise, this should be considered deprecated. We do not have a good alternative to provide at this time with platform specific ownership data for DLC. |
| `characterIds` | array&lt;int64&gt; | A list of the character IDs, for further querying on your part. |
| `seasonHashes` | array&lt;uint32&gt; → DestinySeasonDefinition | A list of seasons that this profile owns. Unlike versionsOwned, these stay with the profile across Platforms, and thus will be valid. It turns out that Stadia Pro subscriptions will give access to seasons but only while playing on Stadia and with an active subscription. So some users (users who have Stadia Pro but choose to play on some other platform) won't see these as available: it will be whatever seasons are available for the platform on which they last played. |
| `seasonPassHashes` | array&lt;uint32&gt; → DestinySeasonPassDefinition | A list of season passes aka reward passes that this profile owns. Unlike versionsOwned, these stay with the profile across Platforms, and thus will be valid. |
| `eventCardHashesOwned` | array&lt;uint32&gt; → DestinyEventCardDefinition | A list of hashes for event cards that a profile owns. Unlike most values in versionsOwned, these stay with the profile across all platforms. |
| `currentSeasonHash` | uint32 → DestinySeasonDefinition? | If populated, this is a reference to the season that is currently active. |
| `currentSeasonPassHash` | uint32 → DestinySeasonPassDefinition? | If populated, this is a reference to the season pass that is currently active. |
| `currentSeasonRewardPowerCap` | int32? | If populated, this is the reward power cap for the current season. |
| `activeEventCardHash` | uint32 → DestinyEventCardDefinition? | If populated, this is a reference to the event card that is currently active. |
| `currentGuardianRank` | int32 → DestinyGuardianRankDefinition | The 'current' Guardian Rank value, which starts at rank 1. This rank value will drop at the start of a new season to your 'renewed' rank from the previous season. |
| `lifetimeHighestGuardianRank` | int32 → DestinyGuardianRankDefinition | The 'lifetime highest' Guardian Rank value, which starts at rank 1. This rank value should never go down. |
| `renewedGuardianRank` | int32 → DestinyGuardianRankDefinition | The seasonal 'renewed' Guardian Rank value. This rank value resets at the start of each new season to the highest-earned non-advanced rank. |

#### Destiny.Definitions.Seasons.DestinyEventCardDefinition

**Object** · *(Manifest definition, table `EventCards`)*

Defines the properties of an 'Event Card' in Destiny 2, to coincide with a seasonal event for additional challenges, premium rewards, a new seal, and a special title. For example: Solstice of Heroes 2022.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `linkRedirectPath` | string | — |
| `color` | Destiny.Misc.DestinyColor | — |
| `images` | Destiny.Definitions.Seasons.DestinyEventCardImages | — |
| `triumphsPresentationNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `sealPresentationNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `eventCardCurrencyList` | array&lt;uint32&gt; → DestinyInventoryItemDefinition | — |
| `ticketCurrencyItemHash` | uint32 → DestinyInventoryItemDefinition | — |
| `ticketVendorHash` | uint32 → DestinyVendorDefinition | — |
| `ticketVendorCategoryHash` | uint32 | — |
| `endTime` | int64 | — |
| `rewardProgressionHash` | uint32 → DestinyProgressionDefinition? | — |
| `weeklyChallengesPresentationNodeHash` | uint32 → DestinyPresentationNodeDefinition? | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Seasons.DestinyEventCardImages

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `unownedCardSleeveImagePath` | string | — |
| `unownedCardSleeveWrapImagePath` | string | — |
| `cardIncompleteImagePath` | string | — |
| `cardCompleteImagePath` | string | — |
| `cardCompleteWrapImagePath` | string | — |
| `progressIconImagePath` | string | — |
| `themeBackgroundImagePath` | string | — |

#### Destiny.Definitions.GuardianRanks.DestinyGuardianRankDefinition

**Object** · *(Manifest definition, table `GuardianRanks`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `rankNumber` | int32 | — |
| `presentationNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `foregroundImagePath` | string | — |
| `overlayImagePath` | string | — |
| `overlayMaskImagePath` | string | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Components.Kiosks.DestinyKiosksComponentDepends on Component "Kiosks"

**Type:** object

A Kiosk is a Vendor (DestinyVendorDefinition) that sells items based on whether you have already acquired that item before. This component returns information about what Kiosk items are available to you on a *Profile* level. It is theoretically possible for Kiosks to have items gated by specific Character as well. If you ever have those, you will find them on the individual character's DestinyCharacterKiosksComponent. Note that, because this component returns vendorItemIndexes (that is to say, indexes into the Kiosk Vendor's itemList property), these results are necessarily content version dependent. Make sure that you have the latest version of the content manifest databases before using this data.

| Property | Type | Description |
| --- | --- | --- |
| `kioskItems` | Mapping&lt;uint32, array&gt; → DestinyVendorDefinition | A dictionary keyed by the Kiosk Vendor's hash identifier (use it to look up the DestinyVendorDefinition for the relevant kiosk vendor), and whose value is a list of all the items that the user can "see" in the Kiosk, and any other interesting metadata. |

#### Destiny.Components.Kiosks.DestinyKioskItem

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `index` | int32 | The index of the item in the related DestinyVendorDefintion's itemList property, representing the sale. |
| `canAcquire` | boolean | If true, the user can not only see the item, but they can acquire it. It is possible that a user can see a kiosk item and not be able to acquire it. |
| `failureIndexes` | array&lt;int32&gt; | Indexes into failureStrings for the Vendor, indicating the reasons why it failed if any. |
| `flavorObjective` | Destiny.Quests.DestinyObjectiveProgress | I may regret naming it this way - but this represents when an item has an objective that doesn't serve a beneficial purpose, but rather is used for "flavor" or additional information. For instance, when Emblems track specific stats, those stats are represented as Objectives on the item. |

#### Destiny.Components.PlugSets.DestinyPlugSetsComponentDepends on Component "ItemSockets"

**Type:** object

Sockets may refer to a "Plug Set": a set of reusable plugs that may be shared across multiple sockets (or even, in theory, multiple sockets over multiple items). This is the set of those plugs that we came across in the users' inventory, along with the values for plugs in the set. Any given set in this component may be represented in Character and Profile-level, as some plugs may be Profile-level restricted, and some character-level restricted. (note that the ones that are even more specific will remain on the actual socket component itself, as they cannot be reused)

| Property | Type | Description |
| --- | --- | --- |
| `plugs` | Mapping&lt;uint32, array&gt; → DestinyPlugSetDefinition | The shared list of plugs for each relevant PlugSet, keyed by the hash identifier of the PlugSet (DestinyPlugSetDefinition). |

#### Destiny.Sockets.DestinyItemPlugBase

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `plugItemHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier of the DestinyInventoryItemDefinition that represents this plug. |
| `canInsert` | boolean | If true, this plug has met all of its insertion requirements. Big if true. |
| `enabled` | boolean | If true, this plug will provide its benefits while inserted. |
| `insertFailIndexes` | array&lt;int32&gt; | If the plug cannot be inserted for some reason, this will have the indexes into the plug item definition's plug.insertionRules property, so you can show the reasons why it can't be inserted. This list will be empty if the plug can be inserted. |
| `enableFailIndexes` | array&lt;int32&gt; | If a plug is not enabled, this will be populated with indexes into the plug item definition's plug.enabledRules property, so that you can show the reasons why it is not enabled. This list will be empty if the plug is enabled. |
| `stackSize` | int32? | If available, this is the stack size to display for the socket plug item. |
| `maxStackSize` | int32? | If available, this is the maximum stack size to display for the socket plug item. |

#### Destiny.Sockets.DestinyItemPlug

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `plugObjectives` | array&lt;Destiny.Quests.DestinyObjectiveProgress&gt; | Sometimes, Plugs may have objectives: these are often used for flavor and display purposes, but they can be used for any arbitrary purpose (both fortunately and unfortunately). Recently (with Season 2) they were expanded in use to be used as the "gating" for whether the plug can be inserted at all. For instance, a Plug might be tracking the number of PVP kills you have made. It will use the parent item's data about that tracking status to determine what to show, and will generally show it using the DestinyObjectiveDefinition's progressDescription property. Refer to the plug's itemHash and objective property for more information if you would like to display even more data. |
| `plugItemHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier of the DestinyInventoryItemDefinition that represents this plug. |
| `canInsert` | boolean | If true, this plug has met all of its insertion requirements. Big if true. |
| `enabled` | boolean | If true, this plug will provide its benefits while inserted. |
| `insertFailIndexes` | array&lt;int32&gt; | If the plug cannot be inserted for some reason, this will have the indexes into the plug item definition's plug.insertionRules property, so you can show the reasons why it can't be inserted. This list will be empty if the plug can be inserted. |
| `enableFailIndexes` | array&lt;int32&gt; | If a plug is not enabled, this will be populated with indexes into the plug item definition's plug.enabledRules property, so that you can show the reasons why it is not enabled. This list will be empty if the plug is enabled. |
| `stackSize` | int32? | If available, this is the stack size to display for the socket plug item. |
| `maxStackSize` | int32? | If available, this is the maximum stack size to display for the socket plug item. |

#### Destiny.Components.Profiles.DestinyProfileProgressionComponentDepends on Component "ProfileProgression"

**Type:** object

The set of progression-related information that applies at a Profile-wide level for your Destiny experience. This differs from the Jimi Hendrix Experience because there's less guitars on fire. Yet. #spoileralert? This will include information such as Checklist info.

| Property | Type | Description |
| --- | --- | --- |
| `checklists` | Mapping&lt;uint32, object&gt; → DestinyChecklistDefinition | The set of checklists that can be examined on a profile-wide basis, keyed by the hash identifier of the Checklist (DestinyChecklistDefinition) For each checklist returned, its value is itself a Dictionary keyed by the checklist's hash identifier with the value being a boolean indicating if it's been discovered yet. |
| `seasonalArtifact` | Destiny.Artifacts.DestinyArtifactProfileScoped | Data related to your progress on the current season's artifact that is the same across characters. |

#### Destiny.Artifacts.DestinyArtifactProfileScoped

**Type:** object

Represents a Seasonal Artifact and all data related to it for the requested Account. It can be combined with Character-scoped data for a full picture of what a character has available/has chosen, or just these settings can be used for overview information.

| Property | Type | Description |
| --- | --- | --- |
| `artifactHash` | uint32 → DestinyArtifactDefinition | — |
| `pointProgression` | Destiny.DestinyProgression | — |
| `pointsAcquired` | int32 | — |
| `powerBonusProgression` | Destiny.DestinyProgression | — |
| `powerBonus` | int32 | — |

#### Destiny.Definitions.Checklists.DestinyChecklistDefinition

**Object** · *(Manifest definition, table `Checklists`)*

By public demand, Checklists are loose sets of "things to do/things you have done" in Destiny that we were actually able to track. They include easter eggs you find in the world, unique chests you unlock, and other such data where the first time you do it is significant enough to be tracked, and you have the potential to "get them all". These may be account-wide, or may be per character. The status of these will be returned in related "Checklist" data coming down from API requests such as GetProfile or GetCharacter. Generally speaking, the items in a checklist can be completed in any order: we return an ordered list which only implies the way we are showing them in our own UI, and you can feel free to alter it as you wish. Note that, in the future, there will be something resembling the old D1 Record Books in at least some vague form. When that is created, it may be that it will supercede much or all of this Checklist data. It remains to be seen if that will be the case, so for now assume that the Checklists will still exist even after the release of D2: Forsaken.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `viewActionString` | string | A localized string prompting you to view the checklist. |
| `scope` | int32 | Indicates whether you will find this checklist on the Profile or Character components. |
| `entries` | array&lt;Destiny.Definitions.Checklists.DestinyChecklistEntryDefinition&gt; | The individual checklist items. Gotta catch 'em all. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Checklists.DestinyChecklistEntryDefinition

**Type:** object

The properties of an individual checklist item. Note that almost everything is optional: it is *highly* variable what kind of data we'll actually be able to return: at times we may have no other relationships to entities at all. Whatever UI you build, do it with the knowledge that any given entry might not actually be able to be associated with some other Destiny entity.

| Property | Type | Description |
| --- | --- | --- |
| `hash` | uint32 | The identifier for this Checklist entry. Guaranteed unique only within this Checklist Definition, and not globally/for all checklists. |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | Even if no other associations exist, we will give you *something* for display properties. In cases where we have no associated entities, it may be as simple as a numerical identifier. |
| `destinationHash` | uint32 → DestinyDestinationDefinition? | — |
| `locationHash` | uint32 → DestinyLocationDefinition? | — |
| `bubbleHash` | uint32? | Note that a Bubble's hash doesn't uniquely identify a "top level" entity in Destiny. Only the combination of location and bubble can uniquely identify a place in the world of Destiny: so if bubbleHash is populated, locationHash must too be populated for it to have any meaning. You can use this property if it is populated to look up the DestinyLocationDefinition's associated .locationReleases[].activityBubbleName property. |
| `activityHash` | uint32 → DestinyActivityDefinition? | — |
| `itemHash` | uint32 → DestinyInventoryItemDefinition? | — |
| `vendorHash` | uint32 → DestinyVendorDefinition? | — |
| `vendorInteractionIndex` | int32? | — |
| `scope` | int32 | The scope at which this specific entry can be computed. |

#### Destiny.Components.Presentation.DestinyPresentationNodesComponentDepends on Component "PresentationNodes"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `nodes` | Mapping&lt;uint32, Destiny.Components.Presentation.DestinyPresentationNodeComponent&gt; → DestinyPresentationNodeDefinition | — |

#### Destiny.Components.Presentation.DestinyPresentationNodeComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `state` | int32 | — |
| `objective` | Destiny.Quests.DestinyObjectiveProgress | An optional property: presentation nodes MAY have objectives, which can be used to infer more human readable data about the progress. However, progressValue and completionValue ought to be considered the canonical values for progress on Progression Nodes. |
| `progressValue` | int32 | How much of the presentation node is considered to be completed so far by the given character/ profile. |
| `completionValue` | int32 | The value at which the presentation node is considered to be completed. |
| `recordCategoryScore` | int32? | If available, this is the current score for the record category that this node represents. |

#### Destiny.DestinyPresentationNodeStateEnumeration

**Enum** (`int32`)

I know this doesn't look like a Flags Enumeration/bitmask right now, but I assure you it is. This is the possible states that a Presentation Node can be in, and it is almost certain that its potential states will increase in the future. So don't treat it like a straight up enumeration.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Invisible` | 1 | If this is set, the game recommends that you not show this node. But you know your life, do what you've got to do. |
| `Obscured` | 2 | Turns out Presentation Nodes can also be obscured. If they are, this is set. |

#### Destiny.Components.Records.DestinyRecordsComponentDepends on Component "Records"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `records` | Mapping&lt;uint32, Destiny.Components.Records.DestinyRecordComponent&gt; | — |
| `recordCategoriesRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | The hash for the root presentation node definition of Triumph categories. |
| `recordSealsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | The hash for the root presentation node definition of Triumph Seals. |

#### Destiny.Components.Records.DestinyRecordComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `state` | int32 | — |
| `objectives` | array&lt;Destiny.Quests.DestinyObjectiveProgress&gt; | — |
| `intervalObjectives` | array&lt;Destiny.Quests.DestinyObjectiveProgress&gt; | — |
| `intervalsRedeemedCount` | int32 | — |
| `completedCount` | int32? | If available, this is the number of times this record has been completed. For example, the number of times a seal title has been gilded. |
| `rewardVisibilty` | array&lt;boolean&gt; | If available, a list that describes which reward rewards should be shown (true) or hidden (false). This property is for regular record rewards, and not for interval objective rewards. |

#### Destiny.DestinyRecordStateEnumeration

**Enum** (`int32`)

A Flags enumeration/bitmask where each bit represents a possible state that a Record/Triumph can be in.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | If there are no flags set, the record is in a state where it *could* be redeemed, but it has not been yet. |
| `RecordRedeemed` | 1 | If this is set, the completed record has been redeemed. |
| `RewardUnavailable` | 2 | If this is set, there's a reward available from this Record but it's unavailable for redemption. |
| `ObjectiveNotCompleted` | 4 | If this is set, the objective for this Record has not yet been completed. |
| `Obscured` | 8 | If this is set, the game recommends that you replace the display text of this Record with DestinyRecordDefinition.stateInfo.obscuredDescription. |
| `Invisible` | 16 | If this is set, the game recommends that you not show this record. Do what you will with this recommendation. |
| `EntitlementUnowned` | 32 | If this is set, you can't complete this record because you lack some permission that's required to complete it. |
| `CanEquipTitle` | 64 | If this is set, the record has a title (check DestinyRecordDefinition for title info) and you can equip it. |

#### Destiny.Components.Records.DestinyProfileRecordsComponentDepends on Component "Records"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `score` | int32 | Your 'active' Triumphs score, maintained for backwards compatibility. |
| `activeScore` | int32 | Your 'active' Triumphs score. |
| `legacyScore` | int32 | Your 'legacy' Triumphs score. |
| `lifetimeScore` | int32 | Your 'lifetime' Triumphs score. |
| `trackedRecordHash` | uint32 → DestinyRecordDefinition? | If this profile is tracking a record, this is the hash identifier of the record it is tracking. |
| `records` | Mapping&lt;uint32, Destiny.Components.Records.DestinyRecordComponent&gt; | — |
| `recordCategoriesRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | The hash for the root presentation node definition of Triumph categories. |
| `recordSealsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | The hash for the root presentation node definition of Triumph Seals. |

#### Destiny.Components.Collectibles.DestinyCollectiblesComponentDepends on Component "Collectibles"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `collectibles` | Mapping&lt;uint32, Destiny.Components.Collectibles.DestinyCollectibleComponent&gt; → DestinyCollectibleDefinition | — |
| `collectionCategoriesRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | The hash for the root presentation node definition of Collection categories. |
| `collectionBadgesRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | The hash for the root presentation node definition of Collection Badges. |

#### Destiny.Components.Collectibles.DestinyCollectibleComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `state` | int32 | — |

#### Destiny.DestinyCollectibleStateEnumeration

**Enum** (`int32`)

A Flags Enumeration/bitmask where each bit represents a different state that the Collectible can be in. A collectible can be in any number of these states, and you can choose to use or ignore any or all of them when making your own UI that shows Collectible info. Our displays are going to honor them, but we're also the kind of people who only pretend to inhale before quickly passing it to the left. So, you know, do what you got to do. (All joking aside, please note the caveat I mention around the Invisible flag: there are cases where it is in the best interest of your users to honor these flags even if you're a "show all the data" person. Collector-oriented compulsion is a very unfortunate and real thing, and I would hate to instill that compulsion in others through showing them items that they cannot earn. Please consider this when you are making your own apps/sites.)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `NotAcquired` | 1 | If this flag is set, you have not yet obtained this collectible. |
| `Obscured` | 2 | If this flag is set, the item is "obscured" to you: you can/should use the alternate item hash found in DestinyCollectibleDefinition.stateInfo.obscuredOverrideItemHash when displaying this collectible instead of the default display info. |
| `Invisible` | 4 | If this flag is set, the collectible should not be shown to the user. Please do consider honoring this flag. It is used - for example - to hide items that a person didn't get from the Eververse. I can't prevent these from being returned in definitions, because some people may have acquired them and thus they should show up: but I would hate for people to start feeling some variant of a Collector's Remorse about these items, and thus increasing their purchasing based on that compulsion. That would be a very unfortunate outcome, and one that I wouldn't like to see happen. So please, whether or not I'm your mom, consider honoring this flag and don't show people invisible collectibles. |
| `CannotAffordMaterialRequirements` | 8 | If this flag is set, the collectible requires payment for creating an instance of the item, and you are lacking in currency. Bring the benjamins next time. Or spinmetal. Whatever. |
| `InventorySpaceUnavailable` | 16 | If this flag is set, you can't pull this item out of your collection because there's no room left in your inventory. |
| `UniquenessViolation` | 32 | If this flag is set, you already have one of these items and can't have a second one. |
| `PurchaseDisabled` | 64 | If this flag is set, the ability to pull this item out of your collection has been disabled. |

#### Destiny.Components.Collectibles.DestinyProfileCollectiblesComponentDepends on Component "Collectibles"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `recentCollectibleHashes` | array&lt;uint32&gt; → DestinyCollectibleDefinition | The list of collectibles determined by the game as having been "recently" acquired. |
| `newnessFlaggedCollectibleHashes` | array&lt;uint32&gt; → DestinyCollectibleDefinition | The list of collectibles determined by the game as having been "recently" acquired. The game client itself actually controls this data, so I personally question whether anyone will get much use out of this: because we can't edit this value through the API. But in case anyone finds it useful, here it is. |
| `collectibles` | Mapping&lt;uint32, Destiny.Components.Collectibles.DestinyCollectibleComponent&gt; → DestinyCollectibleDefinition | — |
| `collectionCategoriesRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | The hash for the root presentation node definition of Collection categories. |
| `collectionBadgesRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | The hash for the root presentation node definition of Collection Badges. |

#### Destiny.Components.Profiles.DestinyProfileTransitoryComponentDepends on Component "Transitory"

**Type:** object

This is an experimental set of data that Bungie considers to be "transitory" - information that may be useful for API users, but that is coming from a non-authoritative data source about information that could potentially change at a more frequent pace than Bungie.net will receive updates about it. This information is provided exclusively for convenience should any of it be useful to users: we provide no guarantees to the accuracy or timeliness of data that comes from this source. Know that this data can potentially be out-of-date or even wrong entirely if the user disconnected from the game or suddenly changed their status before we can receive refreshed data.

| Property | Type | Description |
| --- | --- | --- |
| `partyMembers` | array&lt;Destiny.Components.Profiles.DestinyProfileTransitoryPartyMember&gt; | If you have any members currently in your party, this is some (very) bare-bones information about those members. |
| `currentActivity` | Destiny.Components.Profiles.DestinyProfileTransitoryCurrentActivity | If you are in an activity, this is some transitory info about the activity currently being played. |
| `joinability` | Destiny.Components.Profiles.DestinyProfileTransitoryJoinability | Information about whether and what might prevent you from joining this person on a fireteam. |
| `tracking` | array&lt;Destiny.Components.Profiles.DestinyProfileTransitoryTrackingEntry&gt; | Information about tracked entities. |
| `lastOrbitedDestinationHash` | uint32 → DestinyDestinationDefinition? | The hash identifier for the DestinyDestinationDefinition of the last location you were orbiting when in orbit. |

#### Destiny.Components.Profiles.DestinyProfileTransitoryPartyMember

**Type:** object

This is some bare minimum information about a party member in a Fireteam. Unfortunately, without great computational expense on our side we can only get at the data contained here. I'd like to give you a character ID for example, but we don't have it. But we do have these three pieces of information. May they help you on your quest to show meaningful data about current Fireteams. Notably, we don't and can't feasibly return info on characters. If you can, try to use just the data below for your UI and purposes. Only hit us with further queries if you absolutely must know the character ID of the currently playing character. Pretty please with sugar on top.

| Property | Type | Description |
| --- | --- | --- |
| `membershipId` | int64 | The Membership ID that matches the party member. |
| `emblemHash` | uint32 → DestinyInventoryItemDefinition | The identifier for the DestinyInventoryItemDefinition of the player's emblem. |
| `displayName` | string | The player's last known display name. |
| `status` | int32 | A Flags Enumeration value indicating the states that the player is in relevant to being on a fireteam. |

#### Destiny.DestinyPartyMemberStatesEnumeration

**Enum** (`int32`)

A flags enumeration that represents a Fireteam Member's status.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `FireteamMember` | 1 | This one's pretty obvious - they're on your Fireteam. |
| `PosseMember` | 2 | I don't know what it means to be in a 'Posse', but apparently this is it. |
| `GroupMember` | 4 | Nor do I understand the difference between them being in a 'Group' vs. a 'Fireteam'. I'll update these docs once I get more info. If I get more info. If you're reading this, I never got more info. You're on your own, kid. |
| `PartyLeader` | 8 | This person is the party leader. |

#### Destiny.Components.Profiles.DestinyProfileTransitoryCurrentActivity

**Type:** object

If you are playing in an activity, this is some information about it. Note that we cannot guarantee any of this resembles what ends up in the PGCR in any way. They are sourced by two entirely separate systems with their own logic, and the one we source this data from should be considered non-authoritative in comparison.

| Property | Type | Description |
| --- | --- | --- |
| `startTime` | date-time? | When the activity started. |
| `endTime` | date-time? | If you're still in it but it "ended" (like when folks are dancing around the loot after they beat a boss), this is when the activity ended. |
| `score` | float | This is what our non-authoritative source thought the score was. |
| `highestOpposingFactionScore` | float | If you have human opponents, this is the highest opposing team's score. |
| `numberOfOpponents` | int32 | This is how many human or poorly crafted aimbot opponents you have. |
| `numberOfPlayers` | int32 | This is how many human or poorly crafted aimbots are on your team. |

#### Destiny.Components.Profiles.DestinyProfileTransitoryJoinability

**Type:** object

Some basic information about whether you can be joined, how many slots are left etc. Note that this can change quickly, so it may not actually be useful. But perhaps it will be in some use cases?

| Property | Type | Description |
| --- | --- | --- |
| `openSlots` | int32 | The number of slots still available on this person's fireteam. |
| `privacySetting` | int32 | Who the person is currently allowing invites from. |
| `closedReasons` | int32 | Reasons why a person can't join this person's fireteam. |

#### Destiny.DestinyGamePrivacySettingEnumeration

**Enum** (`int32`)

A player can choose to restrict requests to join their Fireteam to specific states. These are the possible states a user can choose.

| Value | # | Description |
| --- | --- | --- |
| `Open` | 0 | — |
| `ClanAndFriendsOnly` | 1 | — |
| `FriendsOnly` | 2 | — |
| `InvitationOnly` | 3 | — |
| `Closed` | 4 | — |

#### Destiny.DestinyJoinClosedReasonsEnumeration

**Enum** (`int32`)

A Flags enumeration representing the reasons why a person can't join this user's fireteam.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `InMatchmaking` | 1 | The user is currently in matchmaking. |
| `Loading` | 2 | The user is currently in a loading screen. |
| `SoloMode` | 4 | The user is in an activity that requires solo play. |
| `InternalReasons` | 8 | The user can't be joined for one of a variety of internal reasons. Basically, the game can't let you join at this time, but for reasons that aren't under the control of this user. |
| `DisallowedByGameState` | 16 | The user's current activity/quest/other transitory game state is preventing joining. |
| `Offline` | 32768 | The user appears to be offline. |

#### Destiny.Components.Profiles.DestinyProfileTransitoryTrackingEntry

**Type:** object

This represents a single "thing" being tracked by the player. This can point to many types of entities, but only a subset of them will actually have a valid hash identifier for whatever it is being pointed to. It's up to you to interpret what it means when various combinations of these entries have values being tracked.

| Property | Type | Description |
| --- | --- | --- |
| `locationHash` | uint32 → DestinyLocationDefinition? | OPTIONAL - If this is tracking a DestinyLocationDefinition, this is the identifier for that location. |
| `itemHash` | uint32 → DestinyInventoryItemDefinition? | OPTIONAL - If this is tracking the status of a DestinyInventoryItemDefinition, this is the identifier for that item. |
| `objectiveHash` | uint32 → DestinyObjectiveDefinition? | OPTIONAL - If this is tracking the status of a DestinyObjectiveDefinition, this is the identifier for that objective. |
| `activityHash` | uint32 → DestinyActivityDefinition? | OPTIONAL - If this is tracking the status of a DestinyActivityDefinition, this is the identifier for that activity. |
| `questlineItemHash` | uint32 → DestinyInventoryItemDefinition? | OPTIONAL - If this is tracking the status of a quest, this is the identifier for the DestinyInventoryItemDefinition that containst that questline data. |
| `trackedDate` | date-time? | OPTIONAL - I've got to level with you, I don't really know what this is. Is it when you started tracking it? Is it only populated for tracked items that have time limits? I don't know, but we can get at it - when I get time to actually test what it is, I'll update this. In the meantime, bask in the mysterious data. |

#### Destiny.Components.Metrics.DestinyMetricsComponentDepends on Component "Metrics"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `metrics` | Mapping&lt;uint32, Destiny.Components.Metrics.DestinyMetricComponent&gt; | — |
| `metricsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |

#### Destiny.Components.Metrics.DestinyMetricComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `invisible` | boolean | — |
| `objectiveProgress` | Destiny.Quests.DestinyObjectiveProgress | — |

#### Destiny.Components.StringVariables.DestinyStringVariablesComponentDepends on Component "StringVariables"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `integerValuesByHash` | Mapping&lt;uint32, int32&gt; | — |

#### Destiny.Components.Social.DestinySocialCommendationsComponentDepends on Component "SocialCommendations"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `totalScore` | int32 | — |
| `commendationNodePercentagesByHash` | Mapping&lt;uint32, uint32&gt; | The percentage for each commendation type out of total received |
| `scoreDetailValues` | array&lt;int32&gt; | — |
| `commendationNodeScoresByHash` | Mapping&lt;uint32, int32&gt; → DestinySocialCommendationNodeDefinition | — |
| `commendationScoresByHash` | Mapping&lt;uint32, int32&gt; → DestinySocialCommendationDefinition | — |

#### Destiny.Definitions.Social.DestinySocialCommendationNodeDefinition

**Object** · *(Manifest definition, table `SocialCommendationNodes`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `color` | Destiny.Misc.DestinyColor | The color associated with this group of commendations. |
| `tintedIcon` | string | A version of the displayProperties icon tinted with the color of this node. |
| `parentCommendationNodeHash` | uint32 → DestinySocialCommendationNodeDefinition | — |
| `childCommendationNodeHashes` | array&lt;uint32&gt; → DestinySocialCommendationNodeDefinition | A list of hashes that map to child commendation nodes. Only the root commendations node is expected to have child nodes. |
| `childCommendationHashes` | array&lt;uint32&gt; → DestinySocialCommendationDefinition | A list of hashes that map to child commendations. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Social.DestinySocialCommendationDefinition

**Object** · *(Manifest definition, table `SocialCommendations`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `cardImagePath` | string | — |
| `color` | Destiny.Misc.DestinyColor | — |
| `displayPriority` | int32 | — |
| `activityGivingLimit` | int32 | — |
| `parentCommendationNodeHash` | uint32 → DestinySocialCommendationNodeDefinition | — |
| `displayActivities` | array&lt;Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition&gt; | The display properties for the the activities that this commendation is available in. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Entities.Characters.DestinyCharacterComponentDepends on Component "Characters"

**Type:** object

This component contains base properties of the character. You'll probably want to always request this component, but hey you do you.

| Property | Type | Description |
| --- | --- | --- |
| `membershipId` | int64 | Every Destiny Profile has a membershipId. This is provided on the character as well for convenience. |
| `membershipType` | int32 | membershipType tells you the platform on which the character plays. Examine the BungieMembershipType enumeration for possible values. |
| `characterId` | int64 | The unique identifier for the character. |
| `dateLastPlayed` | date-time | The last date that the user played Destiny. |
| `minutesPlayedThisSession` | int64 | If the user is currently playing, this is how long they've been playing. |
| `minutesPlayedTotal` | int64 | If this value is 525,600, then they played Destiny for a year. Or they're a very dedicated Rent fan. Note that this includes idle time, not just time spent actually in activities shooting things. |
| `light` | int32 | The user's calculated "Light Level". Light level is an indicator of your power that mostly matters in the end game, once you've reached the maximum character level: it's a level that's dependent on the average Attack/Defense power of your items. |
| `stats` | Mapping&lt;uint32, int32&gt; | Your character's stats, such as Agility, Resilience, etc... *not* historical stats. You'll have to call a different endpoint for those. |
| `raceHash` | uint32 → DestinyRaceDefinition | Use this hash to look up the character's DestinyRaceDefinition. |
| `genderHash` | uint32 → DestinyGenderDefinition | Use this hash to look up the character's DestinyGenderDefinition. |
| `classHash` | uint32 → DestinyClassDefinition | Use this hash to look up the character's DestinyClassDefinition. |
| `raceType` | int32 | Mostly for historical purposes at this point, this is an enumeration for the character's race. It'll be preferable in the general case to look up the related definition: but for some people this was too convenient to remove. |
| `classType` | int32 | Mostly for historical purposes at this point, this is an enumeration for the character's class. It'll be preferable in the general case to look up the related definition: but for some people this was too convenient to remove. |
| `genderType` | int32 | Mostly for historical purposes at this point, this is an enumeration for the character's Gender. It'll be preferable in the general case to look up the related definition: but for some people this was too convenient to remove. And yeah, it's an enumeration and not a boolean. Fight me. |
| `emblemPath` | string | A shortcut path to the user's currently equipped emblem image. If you're just showing summary info for a user, this is more convenient than examining their equipped emblem and looking up the definition. |
| `emblemBackgroundPath` | string | A shortcut path to the user's currently equipped emblem background image. If you're just showing summary info for a user, this is more convenient than examining their equipped emblem and looking up the definition. |
| `emblemHash` | uint32 → DestinyInventoryItemDefinition | The hash of the currently equipped emblem for the user. Can be used to look up the DestinyInventoryItemDefinition. |
| `emblemColor` | Destiny.Misc.DestinyColor | A shortcut for getting the background color of the user's currently equipped emblem without having to do a DestinyInventoryItemDefinition lookup. |
| `levelProgression` | Destiny.DestinyProgression | The progression that indicates your character's level. Not their light level, but their character level: you know, the thing you max out a couple hours in and then ignore for the sake of light level. |
| `baseCharacterLevel` | int32 | The "base" level of your character, not accounting for any light level. |
| `percentToNextLevel` | float | A number between 0 and 100, indicating the whole and fractional % remaining to get to the next character level. |
| `titleRecordHash` | uint32 → DestinyRecordDefinition? | If this Character has a title assigned to it, this is the identifier of the DestinyRecordDefinition that has that title information. |

#### Destiny.DestinyRaceEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Human` | 0 | — |
| `Awoken` | 1 | — |
| `Exo` | 2 | — |
| `Unknown` | 3 | — |

#### Destiny.Definitions.DestinyRaceDefinition

**Object** · *(Manifest definition, table `Races`)*

In Destiny, "Races" are really more like "Species". Sort of. I mean, are the Awoken a separate species from humans? I'm not sure. But either way, they're defined here. You'll see Exo, Awoken, and Human as examples of these Species. Players will choose one for their character.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `raceType` | int32 | An enumeration defining the existing, known Races/Species for player characters. This value will be the enum value matching this definition. |
| `genderedRaceNames` | Mapping&lt;int32, string&gt; | A localized string referring to the singular form of the Race's name when referred to in gendered form. Keyed by the DestinyGender. |
| `genderedRaceNamesByGenderHash` | Mapping&lt;uint32, string&gt; → DestinyGenderDefinition | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Components.Loadouts.DestinyLoadoutsComponentDepends on Component "CharacterLoadouts"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `loadouts` | array&lt;Destiny.Components.Loadouts.DestinyLoadoutComponent&gt; | — |

#### Destiny.Components.Loadouts.DestinyLoadoutComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `colorHash` | uint32 → DestinyLoadoutColorDefinition | — |
| `iconHash` | uint32 → DestinyLoadoutIconDefinition | — |
| `nameHash` | uint32 → DestinyLoadoutNameDefinition | — |
| `items` | array&lt;Destiny.Components.Loadouts.DestinyLoadoutItemComponent&gt; | — |

#### Destiny.Components.Loadouts.DestinyLoadoutItemComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `itemInstanceId` | int64 | — |
| `plugItemHashes` | array&lt;uint32&gt; → DestinyInventoryItemDefinition | — |

#### Destiny.Definitions.Loadouts.DestinyLoadoutColorDefinition

**Object** · *(Manifest definition, table `LoadoutColors`)*

| Property | Type | Description |
| --- | --- | --- |
| `colorImagePath` | string | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Loadouts.DestinyLoadoutIconDefinition

**Object** · *(Manifest definition, table `LoadoutIcons`)*

| Property | Type | Description |
| --- | --- | --- |
| `iconImagePath` | string | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Loadouts.DestinyLoadoutNameDefinition

**Object** · *(Manifest definition, table `LoadoutNames`)*

| Property | Type | Description |
| --- | --- | --- |
| `name` | string | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Entities.Characters.DestinyCharacterProgressionComponentDepends on Component "CharacterProgressions"

**Type:** object

This component returns anything that could be considered "Progression" on a user: data where the user is gaining levels, reputation, completions, rewards, etc...

| Property | Type | Description |
| --- | --- | --- |
| `progressions` | Mapping&lt;uint32, Destiny.DestinyProgression&gt; → DestinyProgressionDefinition | A Dictionary of all known progressions for the Character, keyed by the Progression's hash. Not all progressions have user-facing data, but those who do will have that data contained in the DestinyProgressionDefinition. |
| `factions` | Mapping&lt;uint32, Destiny.Progression.DestinyFactionProgression&gt; → DestinyFactionDefinition | A dictionary of all known Factions, keyed by the Faction's hash. It contains data about this character's status with the faction. |
| `milestones` | Mapping&lt;uint32, Destiny.Milestones.DestinyMilestone&gt; → DestinyMilestoneDefinition | Milestones are related to the simple progressions shown in the game, but return additional and hopefully helpful information for users about the specifics of the Milestone's status. |
| `quests` | array&lt;Destiny.Quests.DestinyQuestStatus&gt; | If the user has any active quests, the quests' statuses will be returned here. Note that quests have been largely supplanted by Milestones, but that doesn't mean that they won't make a comeback independent of milestones at some point. (Fun fact: quests came back as I feared they would, but we never looped back to populate this... I'm going to put that in the backlog.) |
| `uninstancedItemObjectives` | Mapping&lt;uint32, array&gt; → DestinyInventoryItemDefinition | Sometimes, you have items in your inventory that don't have instances, but still have Objective information. This provides you that objective information for uninstanced items. This dictionary is keyed by the item's hash: which you can use to look up the name and description for the overall task(s) implied by the objective. The value is the list of objectives for this item, and their statuses. |
| `uninstancedItemPerks` | Mapping&lt;uint32, Destiny.Entities.Items.DestinyItemPerksComponent&gt; → DestinyInventoryItemDefinition | Sometimes, you have items in your inventory that don't have instances, but still have perks (for example: Trials passage cards). This gives you the perk information for uninstanced items. This dictionary is keyed by item hash, which you can use to look up the corresponding item definition. The value is the list of perks states for the item. |
| `checklists` | Mapping&lt;uint32, object&gt; → DestinyChecklistDefinition | The set of checklists that can be examined for this specific character, keyed by the hash identifier of the Checklist (DestinyChecklistDefinition) For each checklist returned, its value is itself a Dictionary keyed by the checklist's hash identifier with the value being a boolean indicating if it's been discovered yet. |
| `seasonalArtifact` | Destiny.Artifacts.DestinyArtifactCharacterScoped | Data related to your progress on the current season's artifact that can vary per character. |

#### Destiny.Progression.DestinyFactionProgression

**Type:** object

Mostly for historical purposes, we segregate Faction progressions from other progressions. This is just a DestinyProgression with a shortcut for finding the DestinyFactionDefinition of the faction related to the progression.

| Property | Type | Description |
| --- | --- | --- |
| `factionHash` | uint32 → DestinyFactionDefinition | The hash identifier of the Faction related to this progression. Use it to look up the DestinyFactionDefinition for more rendering info. |
| `factionVendorIndex` | int32 | The index of the Faction vendor that is currently available. Will be set to -1 if no vendors are available. |
| `progressionHash` | uint32 → DestinyProgressionDefinition | The hash identifier of the Progression in question. Use it to look up the DestinyProgressionDefinition in static data. |
| `dailyProgress` | int32 | The amount of progress earned today for this progression. |
| `dailyLimit` | int32 | If this progression has a daily limit, this is that limit. |
| `weeklyProgress` | int32 | The amount of progress earned toward this progression in the current week. |
| `weeklyLimit` | int32 | If this progression has a weekly limit, this is that limit. |
| `currentProgress` | int32 | This is the total amount of progress obtained overall for this progression (for instance, the total amount of Character Level experience earned) |
| `level` | int32 | This is the level of the progression (for instance, the Character Level). |
| `levelCap` | int32 | This is the maximum possible level you can achieve for this progression (for example, the maximum character level obtainable) |
| `stepIndex` | int32 | Progressions define their levels in "steps". Since the last step may be repeatable, the user may be at a higher level than the actual Step achieved in the progression. Not necessarily useful, but potentially interesting for those cruising the API. Relate this to the "steps" property of the DestinyProgression to see which step the user is on, if you care about that. (Note that this is Content Version dependent since it refers to indexes.) |
| `progressToNextLevel` | int32 | The amount of progression (i.e. "Experience") needed to reach the next level of this Progression. Jeez, progression is such an overloaded word. |
| `nextLevelAt` | int32 | The total amount of progression (i.e. "Experience") needed in order to reach the next level. |
| `currentResetCount` | int32? | The number of resets of this progression you've executed this season, if applicable to this progression. |
| `seasonResets` | array&lt;Destiny.DestinyProgressionResetEntry&gt; | Information about historical resets of this progression, if there is any data for it. |
| `rewardItemStates` | array&lt;int32&gt; | Information about historical rewards for this progression, if there is any data for it. |
| `rewardItemSocketOverrideStates` | Mapping&lt;int32, Destiny.DestinyProgressionRewardItemSocketOverrideState&gt; | Information about items stats and states that have socket overrides, if there is any data for it. |

#### Destiny.Milestones.DestinyMilestone

**Type:** object

Represents a runtime instance of a user's milestone status. Live Milestone data should be combined with DestinyMilestoneDefinition data to show the user a picture of what is available for them to do in the game, and their status in regards to said "things to do." Consider it a big, wonky to-do list, or Advisors 3.0 for those who remember the Destiny 1 API.

| Property | Type | Description |
| --- | --- | --- |
| `milestoneHash` | uint32 → DestinyMilestoneDefinition | The unique identifier for the Milestone. Use it to look up the DestinyMilestoneDefinition, so you can combine the other data in this contract with static definition data. |
| `availableQuests` | array&lt;Destiny.Milestones.DestinyMilestoneQuest&gt; | Indicates what quests are available for this Milestone. Usually this will be only a single Quest, but some quests have multiple available that you can choose from at any given time. All possible quests for a milestone can be found in the DestinyMilestoneDefinition, but they must be combined with this Live data to determine which one(s) are actually active right now. It is possible for Milestones to not have any quests. |
| `activities` | array&lt;Destiny.Milestones.DestinyMilestoneChallengeActivity&gt; | The currently active Activities in this milestone, when the Milestone is driven by Challenges. Not all Milestones have Challenges, but when they do this will indicate the Activities and Challenges under those Activities related to this Milestone. |
| `values` | Mapping&lt;string, float&gt; | Milestones may have arbitrary key/value pairs associated with them, for data that users will want to know about but that doesn't fit neatly into any of the common components such as Quests. A good example of this would be - if this existed in Destiny 1 - the number of wins you currently have on your Trials of Osiris ticket. Looking in the DestinyMilestoneDefinition, you can use the string identifier of this dictionary to look up more info about the value, including localized string content for displaying the value. The value in the dictionary is the floating point number. The definition will tell you how to format this number. |
| `vendorHashes` | array&lt;uint32&gt; → DestinyVendorDefinition | A milestone may have one or more active vendors that are "related" to it (that provide rewards, or that are the initiators of the Milestone). I already regret this, even as I'm typing it. [I told you I'd regret this] You see, sometimes a milestone may be directly correlated with a set of vendors that provide varying tiers of rewards. The player may not be able to interact with one or more of those vendors. This will return the hashes of the Vendors that the player *can* interact with, allowing you to show their current inventory as rewards or related items to the Milestone or its activities. Before we even use it, it's already deprecated! How much of a bummer is that? We need more data. |
| `vendors` | array&lt;Destiny.Milestones.DestinyMilestoneVendor&gt; | Replaces vendorHashes, which I knew was going to be trouble the day it walked in the door. This will return not only what Vendors are active and relevant to the activity (in an implied order that you can choose to ignore), but also other data - for example, if the Vendor is featuring a specific item relevant to this event that you should show with them. |
| `rewards` | array&lt;Destiny.Milestones.DestinyMilestoneRewardCategory&gt; | If the entity to which this component is attached has known active Rewards for the player, this will detail information about those rewards, keyed by the RewardEntry Hash. (See DestinyMilestoneDefinition for more information about Reward Entries) Note that these rewards are not for the Quests related to the Milestone. Think of these as "overview/checklist" rewards that may be provided for Milestones that may provide rewards for performing a variety of tasks that aren't under a specific Quest. |
| `startDate` | date-time? | If known, this is the date when the event last began or refreshed. It will only be populated for events with fixed and repeating start and end dates. |
| `endDate` | date-time? | If known, this is the date when the event will next end or repeat. It will only be populated for events with fixed and repeating start and end dates. |
| `order` | int32 | Used for ordering milestones in a display to match how we order them in BNet. May pull from static data, or possibly in the future from dynamic information. |

#### Destiny.Milestones.DestinyMilestoneQuest

**Type:** object

If a Milestone has one or more Quests, this will contain the live information for the character's status with one of those quests.

| Property | Type | Description |
| --- | --- | --- |
| `questItemHash` | uint32 → DestinyInventoryItemDefinition | Quests are defined as Items in content. As such, this is the hash identifier of the DestinyInventoryItemDefinition that represents this quest. It will have pointers to all of the steps in the quest, and display information for the quest (title, description, icon etc) Individual steps will be referred to in the Quest item's DestinyInventoryItemDefinition.setData property, and themselves are Items with their own renderable data. |
| `status` | Destiny.Quests.DestinyQuestStatus | The current status of the quest for the character making the request. |
| `activity` | Destiny.Milestones.DestinyMilestoneActivity | *IF* the Milestone has an active Activity that can give you greater details about what you need to do, it will be returned here. Remember to associate this with the DestinyMilestoneDefinition's activities to get details about the activity, including what specific quest it is related to if you have multiple quests to choose from. |
| `challenges` | array&lt;Destiny.Challenges.DestinyChallengeStatus&gt; | The activities referred to by this quest can have many associated challenges. They are all contained here, with activityHashes so that you can associate them with the specific activity variants in which they can be found. In retrospect, I probably should have put these under the specific Activity Variants, but it's too late to change it now. Theoretically, a quest without Activities can still have Challenges, which is why this is on a higher level than activity/variants, but it probably should have been in both places. That may come as a later revision. |

#### Destiny.Quests.DestinyQuestStatus

**Type:** object

Data regarding the progress of a Quest for a specific character. Quests are composed of multiple steps, each with potentially multiple objectives: this QuestStatus will return Objective data for the *currently active* step in this quest.

| Property | Type | Description |
| --- | --- | --- |
| `questHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier for the Quest Item. (Note: Quests are defined as Items, and thus you would use this to look up the quest's DestinyInventoryItemDefinition). For information on all steps in the quest, you can then examine its DestinyInventoryItemDefinition.setData property for Quest Steps (which are *also* items). You can use the Item Definition to display human readable data about the overall quest. |
| `stepHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier of the current Quest Step, which is also a DestinyInventoryItemDefinition. You can use this to get human readable data about the current step and what to do in that step. |
| `stepObjectives` | array&lt;Destiny.Quests.DestinyObjectiveProgress&gt; | A step can have multiple objectives. This will give you the progress for each objective in the current step, in the order in which they are rendered in-game. |
| `tracked` | boolean | Whether or not the quest is tracked |
| `itemInstanceId` | int64 | The current Quest Step will be an instanced item in the player's inventory. If you care about that, this is the instance ID of that item. |
| `completed` | boolean | Whether or not the whole quest has been completed, regardless of whether or not you have redeemed the rewards for the quest. |
| `redeemed` | boolean | Whether or not you have redeemed rewards for this quest. |
| `started` | boolean | Whether or not you have started this quest. |
| `vendorHash` | uint32? | If the quest has a related Vendor that you should talk to in order to initiate the quest/earn rewards/continue the quest, this will be the hash identifier of that Vendor. Look it up its DestinyVendorDefinition. |

#### Destiny.Milestones.DestinyMilestoneActivity

**Type:** object

Sometimes, we know the specific activity that the Milestone wants you to play. This entity provides additional information about that Activity and all of its variants. (sometimes there's only one variant, but I think you get the point)

| Property | Type | Description |
| --- | --- | --- |
| `activityHash` | uint32 → DestinyActivityDefinition | The hash of an arbitrarily chosen variant of this activity. We'll go ahead and call that the "canonical" activity, because if you're using this value you should only use it for properties that are common across the variants: things like the name of the activity, it's location, etc... Use this hash to look up the DestinyActivityDefinition of this activity for rendering data. |
| `activityModeHash` | uint32 → DestinyActivityModeDefinition? | The hash identifier of the most specific Activity Mode under which this activity is played. This is useful for situations where the activity in question is - for instance - a PVP map, but it's not clear what mode the PVP map is being played under. If it's a playlist, this will be less specific: but hopefully useful in some way. |
| `activityModeType` | int32? | The enumeration equivalent of the most specific Activity Mode under which this activity is played. |
| `modifierHashes` | array&lt;uint32&gt; → DestinyActivityModifierDefinition | If the activity has modifiers, this will be the list of modifiers that all variants have in common. Perform lookups against DestinyActivityModifierDefinition which defines the modifier being applied to get at the modifier data. Note that, in the DestinyActivityDefinition, you will see many more modifiers than this being referred to: those are all *possible* modifiers for the activity, not the active ones. Use only the active ones to match what's really live. |
| `variants` | array&lt;Destiny.Milestones.DestinyMilestoneActivityVariant&gt; | If you want more than just name/location/etc... you're going to have to dig into and show the variants of the conceptual activity. These will differ in seemingly arbitrary ways, like difficulty level and modifiers applied. Show it in whatever way tickles your fancy. |

#### Destiny.Milestones.DestinyMilestoneActivityVariant

**Type:** object

Represents custom data that we know about an individual variant of an activity.

| Property | Type | Description |
| --- | --- | --- |
| `activityHash` | uint32 → DestinyActivityDefinition | The hash for the specific variant of the activity related to this milestone. You can pull more detailed static info from the DestinyActivityDefinition, such as difficulty level. |
| `completionStatus` | Destiny.Milestones.DestinyMilestoneActivityCompletionStatus | An OPTIONAL component: if it makes sense to talk about this activity variant in terms of whether or not it has been completed or what progress you have made in it, this will be returned. Otherwise, this will be NULL. |
| `activityModeHash` | uint32 → DestinyActivityModeDefinition? | The hash identifier of the most specific Activity Mode under which this activity is played. This is useful for situations where the activity in question is - for instance - a PVP map, but it's not clear what mode the PVP map is being played under. If it's a playlist, this will be less specific: but hopefully useful in some way. |
| `activityModeType` | int32? | The enumeration equivalent of the most specific Activity Mode under which this activity is played. |

#### Destiny.Milestones.DestinyMilestoneActivityCompletionStatus

**Type:** object

Represents this player's personal completion status for the Activity under a Milestone, if the activity has trackable completion and progress information. (most activities won't, or the concept won't apply. For instance, it makes sense to talk about a tier of a raid as being Completed or having progress, but it doesn't make sense to talk about a Crucible Playlist in those terms.

| Property | Type | Description |
| --- | --- | --- |
| `completed` | boolean | If the activity has been "completed", that information will be returned here. |
| `phases` | array&lt;Destiny.Milestones.DestinyMilestoneActivityPhase&gt; | If the Activity has discrete "phases" that we can track, that info will be here. Otherwise, this value will be NULL. Note that this is a list and not a dictionary: the order implies the ascending order of phases or progression in this activity. |

#### Destiny.Milestones.DestinyMilestoneActivityPhase

**Type:** object

Represents whatever information we can return about an explicit phase in an activity. In the future, I hope we'll have more than just "guh, you done gone and did something," but for the forseeable future that's all we've got. I'm making it more than just a list of booleans out of that overly-optimistic hope.

| Property | Type | Description |
| --- | --- | --- |
| `complete` | boolean | Indicates if the phase has been completed. |
| `phaseHash` | uint32 | In DestinyActivityDefinition, if the activity has phases, there will be a set of phases defined in the "insertionPoints" property. This is the hash that maps to that phase. |

#### Destiny.Challenges.DestinyChallengeStatus

**Type:** object

Represents the status and other related information for a challenge that is - or was - available to a player. A challenge is a bonus objective, generally tacked onto Quests or Activities, that provide additional variations on play.

| Property | Type | Description |
| --- | --- | --- |
| `objective` | Destiny.Quests.DestinyObjectiveProgress | The progress - including completion status - of the active challenge. |

#### Destiny.Milestones.DestinyMilestoneChallengeActivity

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `activityHash` | uint32 → DestinyActivityDefinition | — |
| `challenges` | array&lt;Destiny.Challenges.DestinyChallengeStatus&gt; | — |
| `modifierHashes` | array&lt;uint32&gt; → DestinyActivityModifierDefinition | If the activity has modifiers, this will be the list of modifiers that all variants have in common. Perform lookups against DestinyActivityModifierDefinition which defines the modifier being applied to get at the modifier data. Note that, in the DestinyActivityDefinition, you will see many more modifiers than this being referred to: those are all *possible* modifiers for the activity, not the active ones. Use only the active ones to match what's really live. |
| `booleanActivityOptions` | Mapping&lt;uint32, boolean&gt; | The set of activity options for this activity, keyed by an identifier that's unique for this activity (not guaranteed to be unique between or across all activities, though should be unique for every *variant* of a given *conceptual* activity: for instance, the original D2 Raid has many variant DestinyActivityDefinitions. While other activities could potentially have the same option hashes, for any given D2 base Raid variant the hash will be unique). As a concrete example of this data, the hashes you get for Raids will correspond to the currently active "Challenge Mode". We don't have any human readable information for these, but savvy 3rd party app users could manually associate the key (a hash identifier for the "option" that is enabled/disabled) and the value (whether it's enabled or disabled presently) On our side, we don't necessarily even know what these are used for (the game designers know, but we don't), and we have no human readable data for them. In order to use them, you will have to do some experimentation. |
| `loadoutRequirementIndex` | int32? | If returned, this is the index into the DestinyActivityDefinition's "loadouts" property, indicating the currently active loadout requirements. |
| `phases` | array&lt;Destiny.Milestones.DestinyMilestoneActivityPhase&gt; | If the Activity has discrete "phases" that we can track, that info will be here. Otherwise, this value will be NULL. Note that this is a list and not a dictionary: the order implies the ascending order of phases or progression in this activity. |

#### Destiny.Milestones.DestinyMilestoneVendor

**Type:** object

If a Milestone has one or more Vendors that are relevant to it, this will contain information about that vendor that you can choose to show.

| Property | Type | Description |
| --- | --- | --- |
| `vendorHash` | uint32 → DestinyVendorDefinition | The hash identifier of the Vendor related to this Milestone. You can show useful things from this, such as thier Faction icon or whatever you might care about. |
| `previewItemHash` | uint32 → DestinyInventoryItemDefinition? | If this vendor is featuring a specific item for this event, this will be the hash identifier of that item. I'm taking bets now on how long we go before this needs to be a list or some other, more complex representation instead and I deprecate this too. I'm going to go with 5 months. Calling it now, 2017-09-14 at 9:46pm PST. |

#### Destiny.Milestones.DestinyMilestoneRewardCategory

**Type:** object

Represents a category of "summary" rewards that can be earned for the Milestone regardless of specific quest rewards that can be earned.

| Property | Type | Description |
| --- | --- | --- |
| `rewardCategoryHash` | uint32 | Look up the relevant DestinyMilestoneDefinition, and then use rewardCategoryHash to look up the category info in DestinyMilestoneDefinition.rewards. |
| `entries` | array&lt;Destiny.Milestones.DestinyMilestoneRewardEntry&gt; | The individual reward entries for this category, and their status. |

#### Destiny.Milestones.DestinyMilestoneRewardEntry

**Type:** object

The character-specific data for a milestone's reward entry. See DestinyMilestoneDefinition for more information about Reward Entries.

| Property | Type | Description |
| --- | --- | --- |
| `rewardEntryHash` | uint32 | The identifier for the reward entry in question. It is important to look up the related DestinyMilestoneRewardEntryDefinition to get the static details about the reward, which you can do by looking up the milestone's DestinyMilestoneDefinition and examining the DestinyMilestoneDefinition.rewards[rewardCategoryHash].rewardEntries[rewardEntryHash] data. |
| `earned` | boolean | If TRUE, the player has earned this reward. |
| `redeemed` | boolean | If TRUE, the player has redeemed/picked up/obtained this reward. Feel free to alias this to "gotTheShinyBauble" in your own codebase. |

#### Destiny.Definitions.Milestones.DestinyMilestoneDefinition

**Object** · *(Manifest definition, table `Milestones`)*

Milestones are an in-game concept where they're attempting to tell you what you can do next in-game. If that sounds a lot like Advisors in Destiny 1, it is! So we threw out Advisors in the Destiny 2 API and tacked all of the data we would have put on Advisors onto Milestones instead. Each Milestone represents something going on in the game right now: - A "ritual activity" you can perform, like nightfall - A "special event" that may have activities related to it, like Taco Tuesday (there's no Taco Tuesday in Destiny 2) - A checklist you can fulfill, like helping your Clan complete all of its weekly objectives - A tutorial quest you can play through, like the introduction to the Crucible. Most of these milestones appear in game as well. Some of them are BNet only, because we're so extra. You're welcome. There are some important caveats to understand about how we currently render Milestones and their deficiencies. The game currently doesn't have any content that actually tells you oughtright *what* the Milestone is: that is to say, what you'll be doing. The best we get is either a description of the overall Milestone, or of the Quest that the Milestone is having you partake in: which is usually something that assumes you already know what it's talking about, like "Complete 5 Challenges". 5 Challenges for what? What's a challenge? These are not questions that the Milestone data will answer for you unfortunately. This isn't great, and in the future I'd like to add some custom text to give you more contextual information to pass on to your users. But for now, you can do what we do to render what little display info we do have: Start by looking at the currently active quest (ideally, you've fetched DestinyMilestone or DestinyPublicMilestone data from the API, so you know the currently active quest for the Milestone in question). Look up the Quests property in the Milestone Definition, and check if it has display properties. If it does, show that as the description of the Milestone. If it doesn't, fall back on the Milestone's description. This approach will let you avoid, whenever possible, the even less useful (and sometimes nonexistant) milestone-level names and descriptions.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `displayPreference` | int32 | A hint to the UI to indicate what to show as the display properties for this Milestone when showing "Live" milestone data. Feel free to show more than this if desired: this hint is meant to simplify our own UI, but it may prove useful to you as well. |
| `image` | string | A custom image someone made just for the milestone. Isn't that special? |
| `milestoneType` | int32 | An enumeration listing one of the possible types of milestones. Check out the DestinyMilestoneType enum for more info! |
| `recruitable` | boolean | If True, then the Milestone has been integrated with BNet's recruiting feature. |
| `friendlyName` | string | If the milestone has a friendly identifier for association with other features - such as Recruiting - that identifier can be found here. This is "friendly" in that it looks better in a URL than whatever the identifier for the Milestone actually is. |
| `showInExplorer` | boolean | If TRUE, this entry should be returned in the list of milestones for the "Explore Destiny" (i.e. new BNet homepage) features of Bungie.net (as long as the underlying event is active) Note that this is a property specifically used by BNet and the companion app for the "Live Events" feature of the front page/welcome view: it's not a reflection of what you see in-game. |
| `showInMilestones` | boolean | Determines whether we'll show this Milestone in the user's personal Milestones list. |
| `explorePrioritizesActivityImage` | boolean | If TRUE, "Explore Destiny" (the front page of BNet and the companion app) prioritize using the activity image over any overriding Quest or Milestone image provided. This unfortunate hack is brought to you by Trials of The Nine. |
| `hasPredictableDates` | boolean | A shortcut for clients - and the server - to understand whether we can predict the start and end dates for this event. In practice, there are multiple ways that an event could have predictable date ranges, but not all events will be able to be predicted via any mechanism (for instance, events that are manually triggered on and off) |
| `quests` | Mapping&lt;uint32, Destiny.Definitions.Milestones.DestinyMilestoneQuestDefinition&gt; | The full set of possible Quests that give the overview of the Milestone event/activity in question. Only one of these can be active at a time for a given Conceptual Milestone, but many of them may be "available" for the user to choose from. (for instance, with Milestones you can choose from the three available Quests, but only one can be active at a time) Keyed by the quest item. As of Forsaken (~September 2018), Quest-style Milestones are being removed for many types of activities. There will likely be further revisions to the Milestone concept in the future. |
| `rewards` | Mapping&lt;uint32, Destiny.Definitions.Milestones.DestinyMilestoneRewardCategoryDefinition&gt; | If this milestone can provide rewards, this will define the categories into which the individual reward entries are placed. This is keyed by the Category's hash, which is only guaranteed to be unique within a given Milestone. |
| `vendorsDisplayTitle` | string | If you're going to show Vendors for the Milestone, you can use this as a localized "header" for the section where you show that vendor data. It'll provide a more context-relevant clue about what the vendor's role is in the Milestone. |
| `vendors` | array&lt;Destiny.Definitions.Milestones.DestinyMilestoneVendorDefinition&gt; | Sometimes, milestones will have rewards provided by Vendors. This definition gives the information needed to understand which vendors are relevant, the order in which they should be returned if order matters, and the conditions under which the Vendor is relevant to the user. |
| `values` | Mapping&lt;string, Destiny.Definitions.Milestones.DestinyMilestoneValueDefinition&gt; | Sometimes, milestones will have arbitrary values associated with them that are of interest to us or to third party developers. This is the collection of those values' definitions, keyed by the identifier of the value and providing useful definition information such as localizable names and descriptions for the value. |
| `isInGameMilestone` | boolean | Some milestones are explicit objectives that you can see and interact with in the game. Some milestones are more conceptual, built by BNet to help advise you on activities and events that happen in-game but that aren't explicitly shown in game as Milestones. If this is TRUE, you can see this as a milestone in the game. If this is FALSE, it's an event or activity you can participate in, but you won't see it as a Milestone in the game's UI. |
| `activities` | array&lt;Destiny.Definitions.Milestones.DestinyMilestoneChallengeActivityDefinition&gt; | A Milestone can now be represented by one or more activities directly (without a backing Quest), and that activity can have many challenges, modifiers, and related to it. |
| `defaultOrder` | int32 | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Milestones.DestinyMilestoneDisplayPreferenceEnumeration

**Enum** (`int32`)

A hint for the UI as to what display information ought to be shown. Defaults to showing the static MilestoneDefinition's display properties. If for some reason the indicated property is not populated, fall back to the MilestoneDefinition.displayProperties.

| Value | # | Description |
| --- | --- | --- |
| `MilestoneDefinition` | 0 | Indicates you should show DestinyMilestoneDefinition.displayProperties for this Milestone. |
| `CurrentQuestSteps` | 1 | Indicates you should show the displayProperties for any currently active Quest Steps in DestinyMilestone.availableQuests. |
| `CurrentActivityChallenges` | 2 | Indicates you should show the displayProperties for any currently active Activities and their Challenges in DestinyMilestone.activities. |

#### Destiny.Definitions.Milestones.DestinyMilestoneTypeEnumeration

**Enum** (`int32`)

The type of milestone. Milestones can be Tutorials, one-time/triggered/non-repeating but not necessarily tutorials, or Repeating Milestones.

| Value | # | Description |
| --- | --- | --- |
| `Unknown` | 0 | — |
| `Tutorial` | 1 | One-time milestones that are specifically oriented toward teaching players about new mechanics and gameplay modes. |
| `OneTime` | 2 | Milestones that, once completed a single time, can never be repeated. |
| `Weekly` | 3 | Milestones that repeat/reset on a weekly basis. They need not all reset on the same day or time, but do need to reset weekly to qualify for this type. |
| `Daily` | 4 | Milestones that repeat or reset on a daily basis. |
| `Special` | 5 | Special indicates that the event is not on a daily/weekly cadence, but does occur more than once. For instance, Iron Banner in Destiny 1 or the Dawning were examples of what could be termed "Special" events. |

#### Destiny.Definitions.Milestones.DestinyMilestoneQuestDefinition

**Type:** object

Any data we need to figure out whether this Quest Item is the currently active one for the conceptual Milestone. Even just typing this description, I already regret it.

| Property | Type | Description |
| --- | --- | --- |
| `questItemHash` | uint32 → DestinyInventoryItemDefinition | The item representing this Milestone quest. Use this hash to look up the DestinyInventoryItemDefinition for the quest to find its steps and human readable data. |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | The individual quests may have different definitions from the overall milestone: if there's a specific active quest, use these displayProperties instead of that of the overall DestinyMilestoneDefinition. |
| `overrideImage` | string | If populated, this image can be shown instead of the generic milestone's image when this quest is live, or it can be used to show a background image for the quest itself that differs from that of the Activity or the Milestone. |
| `questRewards` | Destiny.Definitions.Milestones.DestinyMilestoneQuestRewardsDefinition | The rewards you will get for completing this quest, as best as we could extract them from our data. Sometimes, it'll be a decent amount of data. Sometimes, it's going to be sucky. Sorry. |
| `activities` | Mapping&lt;uint32, Destiny.Definitions.Milestones.DestinyMilestoneActivityDefinition&gt; → DestinyActivityDefinition | The full set of all possible "conceptual activities" that are related to this Milestone. Tiers or alternative modes of play within these conceptual activities will be defined as sub-entities. Keyed by the Conceptual Activity Hash. Use the key to look up DestinyActivityDefinition. |
| `destinationHash` | uint32 → DestinyDestinationDefinition? | Sometimes, a Milestone's quest is related to an entire Destination rather than a specific activity. In that situation, this will be the hash of that Destination. Hotspots are currently the only Milestones that expose this data, but that does not preclude this data from being returned for other Milestones in the future. |

#### Destiny.Definitions.Milestones.DestinyMilestoneQuestRewardsDefinition

**Type:** object

If rewards are given in a quest - as opposed to overall in the entire Milestone - there's way less to track. We're going to simplify this contract as a result. However, this also gives us the opportunity to potentially put more than just item information into the reward data if we're able to mine it out in the future. Remember this if you come back and ask "why are quest reward items nested inside of their own class?"

| Property | Type | Description |
| --- | --- | --- |
| `items` | array&lt;Destiny.Definitions.Milestones.DestinyMilestoneQuestRewardItem&gt; | The items that represent your reward for completing the quest. Be warned, these could be "dummy" items: items that are only used to render a good-looking in- game tooltip, but aren't the actual items themselves. For instance, when experience is given there's often a dummy item representing "experience", with quantity being the amount of experience you got. We don't have a programmatic association between those and whatever Progression is actually getting that experience... yet. |

#### Destiny.Definitions.Milestones.DestinyMilestoneQuestRewardItem

**Type:** object

A subclass of DestinyItemQuantity, that provides not just the item and its quantity but also information that BNet can - at some point - use internally to provide more robust runtime information about the item's qualities. If you want it, please ask! We're just out of time to wire it up right now. Or a clever person just may do it with our existing endpoints.

| Property | Type | Description |
| --- | --- | --- |
| `vendorHash` | uint32 → DestinyVendorDefinition? | The quest reward item *may* be associated with a vendor. If so, this is that vendor. Use this hash to look up the DestinyVendorDefinition. |
| `vendorItemIndex` | int32? | The quest reward item *may* be associated with a vendor. If so, this is the index of the item being sold, which we can use at runtime to find instanced item information for the reward item. |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier for the item in question. Use it to look up the item's DestinyInventoryItemDefinition. |
| `itemInstanceId` | int64? | If this quantity is referring to a specific instance of an item, this will have the item's instance ID. Normally, this will be null. |
| `quantity` | int32 | The amount of the item needed/available depending on the context of where DestinyItemQuantity is being used. |
| `hasConditionalVisibility` | boolean | Indicates that this item quantity may be conditionally shown or hidden, based on various sources of state. For example: server flags, account state, or character progress. |

#### Destiny.Definitions.Milestones.DestinyMilestoneActivityDefinition

**Type:** object

Milestones can have associated activities which provide additional information about the context, challenges, modifiers, state etc... related to this Milestone. Information we need to be able to return that data is defined here, along with Tier data to establish a relationship between a conceptual Activity and its difficulty levels and variants.

| Property | Type | Description |
| --- | --- | --- |
| `conceptualActivityHash` | uint32 → DestinyActivityDefinition | The "Conceptual" activity hash. Basically, we picked the lowest level activity and are treating it as the canonical definition of the activity for rendering purposes. If you care about the specific difficulty modes and variations, use the activities under "Variants". |
| `variants` | Mapping&lt;uint32, Destiny.Definitions.Milestones.DestinyMilestoneActivityVariantDefinition&gt; → DestinyActivityDefinition | A milestone-referenced activity can have many variants, such as Tiers or alternative modes of play. Even if there is only a single variant, the details for these are represented within as a variant definition. It is assumed that, if this DestinyMilestoneActivityDefinition is active, then all variants should be active. If a Milestone could ever split the variants' active status conditionally, they should all have their own DestinyMilestoneActivityDefinition instead! The potential duplication will be worth it for the obviousness of processing and use. |

#### Destiny.Definitions.Milestones.DestinyMilestoneActivityVariantDefinition

**Type:** object

Represents a variant on an activity for a Milestone: a specific difficulty tier, or a specific activity variant for example. These will often have more specific details, such as an associated Guided Game, progression steps, tier-specific rewards, and custom values.

| Property | Type | Description |
| --- | --- | --- |
| `activityHash` | uint32 → DestinyActivityDefinition | The hash to use for looking up the variant Activity's definition (DestinyActivityDefinition), where you can find its distinguishing characteristics such as difficulty level and recommended light level. Frequently, that will be the only distinguishing characteristics in practice, which is somewhat of a bummer. |
| `order` | int32 | If you care to do so, render the variants in the order prescribed by this value. When you combine live Milestone data with the definition, the order becomes more useful because you'll be cross-referencing between the definition and live data. |

#### Destiny.Definitions.Milestones.DestinyMilestoneRewardCategoryDefinition

**Type:** object

The definition of a category of rewards, that contains many individual rewards.

| Property | Type | Description |
| --- | --- | --- |
| `categoryHash` | uint32 | Identifies the reward category. Only guaranteed unique within this specific component! |
| `categoryIdentifier` | string | The string identifier for the category, if you want to use it for some end. Guaranteed unique within the specific component. |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | Hopefully this is obvious by now. |
| `rewardEntries` | Mapping&lt;uint32, Destiny.Definitions.Milestones.DestinyMilestoneRewardEntryDefinition&gt; | If this milestone can provide rewards, this will define the sets of rewards that can be earned, the conditions under which they can be acquired, internal data that we'll use at runtime to determine whether you've already earned or redeemed this set of rewards, and the category that this reward should be placed under. |
| `order` | int32 | If you want to use BNet's recommended order for rendering categories programmatically, use this value and compare it to other categories to determine the order in which they should be rendered. I don't feel great about putting this here, I won't lie. |

#### Destiny.Definitions.Milestones.DestinyMilestoneRewardEntryDefinition

**Type:** object

The definition of a specific reward, which may be contained in a category of rewards and that has optional information about how it is obtained.

| Property | Type | Description |
| --- | --- | --- |
| `rewardEntryHash` | uint32 | The identifier for this reward entry. Runtime data will refer to reward entries by this hash. Only guaranteed unique within the specific Milestone. |
| `rewardEntryIdentifier` | string | The string identifier, if you care about it. Only guaranteed unique within the specific Milestone. |
| `items` | array&lt;Destiny.DestinyItemQuantity&gt; | The items you will get as rewards, and how much of it you'll get. |
| `vendorHash` | uint32 → DestinyVendorDefinition? | If this reward is redeemed at a Vendor, this is the hash of the Vendor to go to in order to redeem the reward. Use this hash to look up the DestinyVendorDefinition. |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | For us to bother returning this info, we should be able to return some kind of information about why these rewards are grouped together. This is ideally that information. Look at how confident I am that this will always remain true. |
| `order` | int32 | If you want to follow BNet's ordering of these rewards, use this number within a given category to order the rewards. Yeah, I know. I feel dirty too. |

#### Destiny.Definitions.Milestones.DestinyMilestoneVendorDefinition

**Type:** object

If the Milestone or a component has vendors whose inventories could/should be displayed that are relevant to it, this will return the vendor in question. It also contains information we need to determine whether that vendor is actually relevant at the moment, given the user's current state.

| Property | Type | Description |
| --- | --- | --- |
| `vendorHash` | uint32 → DestinyVendorDefinition | The hash of the vendor whose wares should be shown as associated with the Milestone. |

#### Destiny.Definitions.Milestones.DestinyMilestoneValueDefinition

**Type:** object

The definition for information related to a key/value pair that is relevant for a particular Milestone or component within the Milestone. This lets us more flexibly pass up information that's useful to someone, even if it's not necessarily us.

| Property | Type | Description |
| --- | --- | --- |
| `key` | string | — |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |

#### Destiny.Definitions.Milestones.DestinyMilestoneChallengeActivityDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `activityHash` | uint32 → DestinyActivityDefinition | The activity for which this challenge is active. |
| `challenges` | array&lt;Destiny.Definitions.Milestones.DestinyMilestoneChallengeDefinition&gt; | — |
| `activityGraphNodes` | array&lt;Destiny.Definitions.Milestones.DestinyMilestoneChallengeActivityGraphNodeEntry&gt; | If the activity and its challenge is visible on any of these nodes, it will be returned. |
| `phases` | array&lt;Destiny.Definitions.Milestones.DestinyMilestoneChallengeActivityPhase&gt; | Phases related to this activity, if there are any. These will be listed in the order in which they will appear in the actual activity. |

#### Destiny.Definitions.Milestones.DestinyMilestoneChallengeDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `challengeObjectiveHash` | uint32 → DestinyObjectiveDefinition | The challenge related to this milestone. |

#### Destiny.Definitions.Milestones.DestinyMilestoneChallengeActivityGraphNodeEntry

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `activityGraphHash` | uint32 | — |
| `activityGraphNodeHash` | uint32 | — |

#### Destiny.Definitions.Milestones.DestinyMilestoneChallengeActivityPhase

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `phaseHash` | uint32 | The hash identifier of the activity's phase. |

#### Destiny.Entities.Items.DestinyItemPerksComponentDepends on Component "ItemPerks"

**Type:** object

Instanced items can have perks: benefits that the item bestows. These are related to DestinySandboxPerkDefinition, and sometimes - but not always - have human readable info. When they do, they are the icons and text that you see in an item's tooltip. Talent Grids, Sockets, and the item itself can apply Perks, which are then summarized here for your convenience.

| Property | Type | Description |
| --- | --- | --- |
| `perks` | array&lt;Destiny.Perks.DestinyPerkReference&gt; | The list of perks to display in an item tooltip - and whether or not they have been activated. |

#### Destiny.Perks.DestinyPerkReference

**Type:** object

The list of perks to display in an item tooltip - and whether or not they have been activated. Perks apply a variety of effects to a character, and are generally either intrinsic to the item or provided in activated talent nodes or sockets.

| Property | Type | Description |
| --- | --- | --- |
| `perkHash` | uint32 → DestinySandboxPerkDefinition | The hash identifier for the perk, which can be used to look up DestinySandboxPerkDefinition if it exists. Be warned, perks frequently do not have user-viewable information. You should examine whether you actually found a name/description in the perk's definition before you show it to the user. |
| `iconPath` | string | The icon for the perk. |
| `isActive` | boolean | Whether this perk is currently active. (We may return perks that you have not actually activated yet: these represent perks that you should show in the item's tooltip, but that the user has not yet activated.) |
| `visible` | boolean | Some perks provide benefits, but aren't visible in the UI. This value will let you know if this is perk should be shown in your UI. |

#### Destiny.Artifacts.DestinyArtifactCharacterScoped

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `artifactHash` | uint32 → DestinyArtifactDefinition | — |
| `pointsUsed` | int32 | — |
| `resetCount` | int32 | — |
| `tiers` | array&lt;Destiny.Artifacts.DestinyArtifactTier&gt; | — |

#### Destiny.Artifacts.DestinyArtifactTier

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `tierHash` | uint32 | — |
| `isUnlocked` | boolean | — |
| `pointsToUnlock` | int32 | — |
| `items` | array&lt;Destiny.Artifacts.DestinyArtifactTierItem&gt; | — |

#### Destiny.Artifacts.DestinyArtifactTierItem

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | — |
| `isActive` | boolean | — |
| `isVisible` | boolean | — |

#### Destiny.Entities.Characters.DestinyCharacterRenderComponentDepends on Component "CharacterRenderData"

**Type:** object

Only really useful if you're attempting to render the character's current appearance in 3D, this returns a bare minimum of information, pre-aggregated, that you'll need to perform that rendering. Note that you need to combine this with other 3D assets and data from our servers. Examine the Javascript returned by <https://bungie.net/sharedbundle/spasm> to see how we use this data, but be warned: the rabbit hole goes pretty deep.

| Property | Type | Description |
| --- | --- | --- |
| `customDyes` | array&lt;Destiny.DyeReference&gt; | Custom dyes, calculated by iterating over the character's equipped items. Useful for pre-fetching all of the dye data needed from our server. |
| `customization` | Destiny.Character.DestinyCharacterCustomization | This is actually something that Spasm.js *doesn't* do right now, and that we don't return assets for yet. This is the data about what character customization options you picked. You can combine this with DestinyCharacterCustomizationOptionDefinition to show some cool info, and hopefully someday to actually render a user's face in 3D. We'll see if we ever end up with time for that. |
| `peerView` | Destiny.Character.DestinyCharacterPeerView | A minimal view of: - Equipped items - The rendering-related custom options on those equipped items Combined, that should be enough to render all of the items on the equipped character. |

#### Destiny.Character.DestinyCharacterCustomization

**Type:** object

Raw data about the customization options chosen for a character's face and appearance. You can look up the relevant class/race/gender combo in DestinyCharacterCustomizationOptionDefinition for the character, and then look up these values within the CustomizationOptions found to pull some data about their choices. Warning: not all of that data is meaningful. Some data has useful icons. Others have nothing, and are only meant for 3D rendering purposes (which we sadly do not expose yet)

| Property | Type | Description |
| --- | --- | --- |
| `personality` | uint32 | — |
| `face` | uint32 | — |
| `skinColor` | uint32 | — |
| `lipColor` | uint32 | — |
| `eyeColor` | uint32 | — |
| `hairColors` | array&lt;uint32&gt; | — |
| `featureColors` | array&lt;uint32&gt; | — |
| `decalColor` | uint32 | — |
| `wearHelmet` | boolean | — |
| `hairIndex` | int32 | — |
| `featureIndex` | int32 | — |
| `decalIndex` | int32 | — |

#### Destiny.Character.DestinyCharacterPeerView

**Type:** object

A minimal view of a character's equipped items, for the purpose of rendering a summary screen or showing the character in 3D.

| Property | Type | Description |
| --- | --- | --- |
| `equipment` | array&lt;Destiny.Character.DestinyItemPeerView&gt; | — |

#### Destiny.Character.DestinyItemPeerView

**Type:** object

Bare minimum summary information for an item, for the sake of 3D rendering the item.

| Property | Type | Description |
| --- | --- | --- |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier of the item in question. Use it to look up the DestinyInventoryItemDefinition of the item for static rendering data. |
| `dyes` | array&lt;Destiny.DyeReference&gt; | The list of dyes that have been applied to this item. |

#### Destiny.Entities.Characters.DestinyCharacterActivitiesComponentDepends on Component "CharacterActivities"

**Type:** object

This component holds activity data for a character. It will tell you about the character's current activity status, as well as activities that are available to the user.

| Property | Type | Description |
| --- | --- | --- |
| `dateActivityStarted` | date-time | The last date that the user started playing an activity. |
| `availableActivities` | array&lt;Destiny.DestinyActivity&gt; | The list of activities that the user can play. |
| `availableActivityInteractables` | array&lt;Destiny.Definitions.FireteamFinder.DestinyActivityInteractableReference&gt; | The list of activity interactables that the player can interact with. |
| `difficultyTierCollections` | Mapping&lt;uint32, Destiny.DestinyActivityDifficultyTierCollectionComponent&gt; | The activity difficulty tier states for this character. |
| `selectableSkullCollections` | Mapping&lt;uint32, Destiny.DestinyActivitySelectableSkullCollectionComponent&gt; | The selectable activity skulls states for this character. |
| `currentActivityHash` | uint32 → DestinyActivityDefinition | If the user is in an activity, this will be the hash of the Activity being played. Note that you must combine this info with currentActivityModeHash to get a real picture of what the user is doing right now. For instance, PVP "Activities" are just maps: it's the ActivityMode that determines what type of PVP game they're playing. |
| `currentActivityModeHash` | uint32 → DestinyActivityModeDefinition | If the user is in an activity, this will be the hash of the activity mode being played. Combine with currentActivityHash to give a person a full picture of what they're doing right now. |
| `currentActivityModeType` | int32? | And the current activity's most specific mode type, if it can be found. |
| `currentActivityModeHashes` | array&lt;uint32&gt; → DestinyActivityModeDefinition | If the user is in an activity, this will be the hashes of the DestinyActivityModeDefinition being played. Combine with currentActivityHash to give a person a full picture of what they're doing right now. |
| `currentActivityModeTypes` | array&lt;int32&gt; | All Activity Modes that apply to the current activity being played, in enum form. |
| `currentPlaylistActivityHash` | uint32 → DestinyActivityDefinition? | If the user is in a playlist, this is the hash identifier for the playlist that they chose. |
| `lastCompletedStoryHash` | uint32 → DestinyActivityDefinition | This will have the activity hash of the last completed story/campaign mission, in case you care about that. |

#### Destiny.DestinyActivity

**Type:** object

Represents the "Live" data that we can obtain about a Character's status with a specific Activity. This will tell you whether the character can participate in the activity, as well as some other basic mutable information. Meant to be combined with static DestinyActivityDefinition data for a full picture of the Activity.

| Property | Type | Description |
| --- | --- | --- |
| `activityHash` | uint32 → DestinyActivityDefinition | The hash identifier of the Activity. Use this to look up the DestinyActivityDefinition of the activity. |
| `isNew` | boolean | If true, then the activity should have a "new" indicator in the Director UI. |
| `canLead` | boolean | If true, the user is allowed to lead a Fireteam into this activity. |
| `canJoin` | boolean | If true, the user is allowed to join with another Fireteam in this activity. |
| `isCompleted` | boolean | If true, we both have the ability to know that the user has completed this activity and they have completed it. Unfortunately, we can't necessarily know this for all activities. As such, this should probably only be used if you already know in advance which specific activities you wish to check. |
| `isVisible` | boolean | If true, the user should be able to see this activity. |
| `displayLevel` | int32? | The difficulty level of the activity, if applicable. |
| `recommendedLight` | int32? | The recommended light level for the activity, if applicable. |
| `difficultyTier` | int32 | A DestinyActivityDifficultyTier enum value indicating the difficulty of the activity. |
| `challenges` | array&lt;Destiny.Challenges.DestinyChallengeStatus&gt; | — |
| `modifierHashes` | array&lt;uint32&gt; → DestinyActivityModifierDefinition | If the activity has modifiers, this will be the list of modifiers that all variants have in common. Perform lookups against DestinyActivityModifierDefinition which defines the modifier being applied to get at the modifier data. Note that, in the DestinyActivityDefinition, you will see many more modifiers than this being referred to: those are all *possible* modifiers for the activity, not the active ones. Use only the active ones to match what's really live. |
| `booleanActivityOptions` | Mapping&lt;uint32, boolean&gt; | The set of activity options for this activity, keyed by an identifier that's unique for this activity (not guaranteed to be unique between or across all activities, though should be unique for every *variant* of a given *conceptual* activity: for instance, the original D2 Raid has many variant DestinyActivityDefinitions. While other activities could potentially have the same option hashes, for any given D2 base Raid variant the hash will be unique). As a concrete example of this data, the hashes you get for Raids will correspond to the currently active "Challenge Mode". We don't have any human readable information for these, but savvy 3rd party app users could manually associate the key (a hash identifier for the "option" that is enabled/disabled) and the value (whether it's enabled or disabled presently) On our side, we don't necessarily even know what these are used for (the game designers know, but we don't), and we have no human readable data for them. In order to use them, you will have to do some experimentation. |
| `loadoutRequirementIndex` | int32? | If returned, this is the index into the DestinyActivityDefinition's "loadouts" property, indicating the currently active loadout requirements. |
| `visibleRewards` | array&lt;Destiny.Definitions.DestinyActivityRewardMapping&gt; | A filtered list of reward mappings with only the currently visible reward items. |
| `isFocusedActivity` | boolean | Whether or not this activity is currently in the "featured" carousel of the Portal |

#### Destiny.DestinyActivityDifficultyTierEnumeration

**Enum** (`int32`)

An enumeration representing the potential difficulty levels of an activity. Their names are... more qualitative than quantitative.

| Value | # | Description |
| --- | --- | --- |
| `Trivial` | 0 | — |
| `Easy` | 1 | — |
| `Normal` | 2 | — |
| `Challenging` | 3 | — |
| `Hard` | 4 | — |
| `Brave` | 5 | — |
| `AlmostImpossible` | 6 | — |
| `Impossible` | 7 | — |

#### Destiny.Definitions.DestinyActivityRewardMapping

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `displayBehavior` | int32 | — |
| `rewardItems` | array&lt;Destiny.Definitions.DestinyActivityRewardItem&gt; | — |

#### Destiny.DestinyActivityRewardDisplayModeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Aggregate` | 0 | — |
| `PickFirst` | 1 | — |
| `Count` | 2 | — |

#### Destiny.Definitions.DestinyActivityRewardItem

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `itemQuantity` | Destiny.DestinyItemQuantity | — |
| `uiStyle` | string | — |

#### Destiny.Definitions.FireteamFinder.DestinyActivityInteractableReference

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `activityInteractableHash` | uint32 → DestinyActivityInteractableDefinition | — |
| `activityInteractableElementIndex` | int32 | — |

#### Destiny.Definitions.Activities.DestinyActivityInteractableDefinition

**Object** · *(Manifest definition, table `ActivityInteractables`)*

There are times in every Activity's life when interacting with an object in the world will result in another Activity activating. Well, not every Activity. Just certain ones. Anyways, this defines a set of interactable components, the activities that they spawn when you interact with them, and the conditions under which they can be interacted with. Sadly, we don't get any *really* good data for them, like positional data... yet. I have hopes for future data that we could put on this.

| Property | Type | Description |
| --- | --- | --- |
| `entries` | array&lt;Destiny.Definitions.Activities.DestinyActivityInteractableEntryDefinition&gt; | The possible interactables in this activity interactable definition. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Activities.DestinyActivityInteractableEntryDefinition

**Type:** object

Defines a specific interactable and the action that can occur when triggered.

| Property | Type | Description |
| --- | --- | --- |
| `activityHash` | uint32 → DestinyActivityDefinition | The activity that will trigger when you interact with this interactable. |

#### Destiny.DestinyActivityDifficultyTierCollectionComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `difficultyTierCollectionHash` | uint32 → DestinyActivityDifficultyTierCollectionDefinition | — |
| `difficultyTiers` | array&lt;Destiny.DestinyActivityDifficultyTierComponent&gt; | — |

#### Destiny.DestinyActivityDifficultyTierComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `difficultyTierIndex` | int32 | — |
| `fixedActivitySkulls` | array&lt;Destiny.DestinyActivitySkullComponent&gt; | — |

#### Destiny.DestinyActivitySkullComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `hash` | uint32 | — |
| `skullIdentifierHash` | uint32 | — |
| `isEnabled` | boolean | — |

#### Destiny.DestinyActivitySelectableSkullCollectionComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `selectableSkullCollectionHash` | uint32 → DestinyActivitySelectableSkullCollectionDefinition | — |
| `selectableSkulls` | array&lt;Destiny.DestinyActivitySkullComponent&gt; | — |

#### Destiny.Entities.Items.DestinyItemObjectivesComponentDepends on Component "ItemObjectives"

**Type:** object

Items can have objectives and progression. When you request this block, you will obtain information about any Objectives and progression tied to this item.

| Property | Type | Description |
| --- | --- | --- |
| `objectives` | array&lt;Destiny.Quests.DestinyObjectiveProgress&gt; | If the item has a hard association with objectives, your progress on them will be defined here. Objectives are our standard way to describe a series of tasks that have to be completed for a reward. |
| `flavorObjective` | Destiny.Quests.DestinyObjectiveProgress | I may regret naming it this way - but this represents when an item has an objective that doesn't serve a beneficial purpose, but rather is used for "flavor" or additional information. For instance, when Emblems track specific stats, those stats are represented as Objectives on the item. |
| `dateCompleted` | date-time? | If we have any information on when these objectives were completed, this will be the date of that completion. This won't be on many items, but could be interesting for some items that do store this information. |

#### Destiny.Components.Records.DestinyCharacterRecordsComponentDepends on Component "Records"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `featuredRecordHashes` | array&lt;uint32&gt; → DestinyRecordDefinition | — |
| `records` | Mapping&lt;uint32, Destiny.Components.Records.DestinyRecordComponent&gt; | — |
| `recordCategoriesRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | The hash for the root presentation node definition of Triumph categories. |
| `recordSealsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | The hash for the root presentation node definition of Triumph Seals. |

#### Destiny.Components.Craftables.DestinyCraftablesComponentDepends on Component "Craftables"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `craftables` | Mapping&lt;uint32, Destiny.Components.Craftables.DestinyCraftableComponent&gt; → DestinyInventoryItemDefinition | A map of craftable item hashes to craftable item state components. |
| `craftingRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | The hash for the root presentation node definition of craftable item categories. |

#### Destiny.Components.Craftables.DestinyCraftableComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `visible` | boolean | — |
| `failedRequirementIndexes` | array&lt;int32&gt; | If the requirements are not met for crafting this item, these will index into the list of failure strings. |
| `sockets` | array&lt;Destiny.Components.Craftables.DestinyCraftableSocketComponent&gt; | Plug item state for the crafting sockets. |

#### Destiny.Components.Craftables.DestinyCraftableSocketComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `plugSetHash` | uint32 → DestinyPlugSetDefinition | — |
| `plugs` | array&lt;Destiny.Components.Craftables.DestinyCraftableSocketPlugComponent&gt; | Unlock state for plugs in the socket plug set definition |

#### Destiny.Components.Craftables.DestinyCraftableSocketPlugComponent

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `plugItemHash` | uint32 → DestinyInventoryItemDefinition | — |
| `failedRequirementIndexes` | array&lt;int32&gt; | Index into the unlock requirements to display failure descriptions |

#### Destiny.Entities.Items.DestinyItemInstanceComponentDepends on Component "ItemInstances"

**Type:** object

If an item is "instanced", this will contain information about the item's instance that doesn't fit easily into other components. One might say this is the "essential" instance data for the item. Items are instanced if they require information or state that can vary. For instance, weapons are Instanced: they are given a unique identifier, uniquely generated stats, and can have their properties altered. Non-instanced items have none of these things: for instance, Glimmer has no unique properties aside from how much of it you own. You can tell from an item's definition whether it will be instanced or not by looking at the DestinyInventoryItemDefinition's definition.inventory.isInstanceItem property.

| Property | Type | Description |
| --- | --- | --- |
| `damageType` | int32 | If the item has a damage type, this is the item's current damage type. |
| `damageTypeHash` | uint32 → DestinyDamageTypeDefinition? | The current damage type's hash, so you can look up localized info and icons for it. |
| `primaryStat` | Destiny.DestinyStat | The item stat that we consider to be "primary" for the item. For instance, this would be "Attack" for Weapons or "Defense" for armor. |
| `itemLevel` | int32 | The Item's "Level" has the most significant bearing on its stats, such as Light and Power. |
| `quality` | int32 | The "Quality" of the item has a lesser - but still impactful - bearing on stats like Light and Power. |
| `isEquipped` | boolean | Is the item currently equipped on the given character? |
| `canEquip` | boolean | If this is an equippable item, you can check it here. There are permanent as well as transitory reasons why an item might not be able to be equipped: check cannotEquipReason for details. |
| `equipRequiredLevel` | int32 | If the item cannot be equipped until you reach a certain level, that level will be reflected here. |
| `unlockHashesRequiredToEquip` | array&lt;uint32&gt; → DestinyUnlockDefinition | Sometimes, there are limitations to equipping that are represented by character-level flags called "unlocks". This is a list of flags that they need in order to equip the item that the character has not met. Use these to look up the descriptions to show in your UI by looking up the relevant DestinyUnlockDefinitions for the hashes. |
| `cannotEquipReason` | int32 | If you cannot equip the item, this is a flags enum that enumerates all of the reasons why you couldn't equip the item. You may need to refine your UI further by using unlockHashesRequiredToEquip and equipRequiredLevel. |
| `breakerType` | int32? | If populated, this item has a breaker type corresponding to the given value. See DestinyBreakerTypeDefinition for more details. |
| `breakerTypeHash` | uint32 → DestinyBreakerTypeDefinition? | If populated, this is the hash identifier for the item's breaker type. See DestinyBreakerTypeDefinition for more details. |
| `energy` | Destiny.Entities.Items.DestinyItemInstanceEnergy | IF populated, this item supports Energy mechanics (i.e. Armor 2.0), and these are the current details of its energy type and available capacity to spend energy points. |
| `gearTier` | int32? | Gear Tier, if applicable, fished up from the unlock value items.gear_tier |

#### Destiny.EquipFailureReasonEnumeration

**Enum** (`int32`)

The reasons why an item cannot be equipped, if any. Many flags can be set, or "None" if

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | The item is/was able to be equipped. |
| `ItemUnequippable` | 1 | This is not the kind of item that can be equipped. Did you try equipping Glimmer or something? |
| `ItemUniqueEquipRestricted` | 2 | This item is part of a "unique set", and you can't have more than one item of that same set type equipped at once. For instance, if you already have an Exotic Weapon equipped, you can't equip a second one in another weapon slot. |
| `ItemFailedUnlockCheck` | 4 | This item has state-based gating that prevents it from being equipped in certain circumstances. For instance, an item might be for Warlocks only and you're a Titan, or it might require you to have beaten some special quest that you haven't beaten yet. Use the additional failure data passed on the item itself to get more information about what the specific failure case was (See DestinyInventoryItemDefinition and DestinyItemInstanceComponent) |
| `ItemFailedLevelCheck` | 8 | This item requires you to have reached a specific character level in order to equip it, and you haven't reached that level yet. |
| `ItemWrapped` | 16 | This item is 'wrapped' and must be unwrapped before being equipped. NOTE: This value used to be called ItemNotOnCharacter but that is no longer accurate. |
| `ItemNotLoaded` | 32 | This item is not yet loaded and cannot be equipped yet. |
| `ItemEquipBlocklisted` | 64 | This item is block-listed and cannot be equipped. |
| `ItemLoadoutRequirementNotMet` | 128 | This item does not meet the loadout requirements for the current activity |

#### Destiny.Entities.Items.DestinyItemInstanceEnergy

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `energyTypeHash` | uint32 → DestinyEnergyTypeDefinition | The type of energy for this item. Plugs that require Energy can only be inserted if they have the "Any" Energy Type or the matching energy type of this item. This is a reference to the DestinyEnergyTypeDefinition for the energy type, where you can find extended info about it. |
| `energyType` | int32 | This is the enum version of the Energy Type value, for convenience. |
| `energyCapacity` | int32 | The total capacity of Energy that the item currently has, regardless of if it is currently being used. |
| `energyUsed` | int32 | The amount of Energy currently in use by inserted plugs. |
| `energyUnused` | int32 | The amount of energy still available for inserting new plugs. |

#### Destiny.Definitions.DestinyUnlockDefinition

**Object** · *(Manifest definition, table `Unlocks`)*

Unlock Flags are small bits (literally, a bit, as in a boolean value) that the game server uses for an extremely wide range of state checks, progress storage, and other interesting tidbits of information.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | Sometimes, but not frequently, these unlock flags also have human readable information: usually when they are being directly tested for some requirement, in which case the string is a localized description of why the requirement check failed. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Entities.Items.DestinyItemRenderComponentDepends on Component "ItemRenderData"

**Type:** object

Many items can be rendered in 3D. When you request this block, you will obtain the custom data needed to render this specific instance of the item.

| Property | Type | Description |
| --- | --- | --- |
| `useCustomDyes` | boolean | If you should use custom dyes on this item, it will be indicated here. |
| `artRegions` | Mapping&lt;int32, int32&gt; | A dictionary for rendering gear components, with: key = Art Arrangement Region Index value = The chosen Arrangement Index for the Region, based on the value of a stat on the item used for making the choice. |

#### Destiny.Entities.Items.DestinyItemStatsComponentDepends on Component "ItemStats"

**Type:** object

If you want the stats on an item's instanced data, get this component. These are stats like Attack, Defense etc... and *not* historical stats. Note that some stats have additional computation in-game at runtime - for instance, Magazine Size - and thus these stats might not be 100% accurate compared to what you see in-game for some stats. I know, it sucks. I hate it too.

| Property | Type | Description |
| --- | --- | --- |
| `stats` | Mapping&lt;uint32, Destiny.DestinyStat&gt; → DestinyStatDefinition | If the item has stats that it provides (damage, defense, etc...), it will be given here. |

#### Destiny.Entities.Items.DestinyItemSocketsComponentDepends on Component "ItemSockets"

**Type:** object

Instanced items can have sockets, which are slots on the item where plugs can be inserted. Sockets are a bit complex: be sure to examine the documentation on the DestinyInventoryItemDefinition's "socket" block and elsewhere on these objects for more details.

| Property | Type | Description |
| --- | --- | --- |
| `sockets` | array&lt;Destiny.Entities.Items.DestinyItemSocketState&gt; | The list of all sockets on the item, and their status information. |

#### Destiny.Entities.Items.DestinyItemSocketState

**Type:** object

The status of a given item's socket. (which plug is inserted, if any: whether it is enabled, what "reusable" plugs can be inserted, etc...) If I had it to do over, this would probably have a DestinyItemPlug representing the inserted item instead of most of these properties. :shrug:

| Property | Type | Description |
| --- | --- | --- |
| `plugHash` | uint32 → DestinyInventoryItemDefinition? | The currently active plug, if any. Note that, because all plugs are statically defined, its effect on stats and perks can be statically determined using the plug item's definition. The stats and perks can be taken at face value on the plug item as the stats and perks it will provide to the user/item. |
| `isEnabled` | boolean | Even if a plug is inserted, it doesn't mean it's enabled. This flag indicates whether the plug is active and providing its benefits. |
| `isVisible` | boolean | A plug may theoretically provide benefits but not be visible - for instance, some older items use a plug's damage type perk to modify their own damage type. These, though they are not visible, still affect the item. This field indicates that state. An invisible plug, while it provides benefits if it is Enabled, cannot be directly modified by the user. |
| `enableFailIndexes` | array&lt;int32&gt; | If a plug is inserted but not enabled, this will be populated with indexes into the plug item definition's plug.enabledRules property, so that you can show the reasons why it is not enabled. |

#### Destiny.Components.Items.DestinyItemReusablePlugsComponentDepends on Component "ItemReusablePlugs"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `plugs` | Mapping&lt;int32, array&gt; | If the item supports reusable plugs, this is the list of plugs that are allowed to be used for the socket, and any relevant information about whether they are "enabled", whether they are allowed to be inserted, and any other information such as objectives. A Reusable Plug is a plug that you can always insert into this socket as long as its insertion rules are passed, regardless of whether or not you have the plug in your inventory. An example of it failing an insertion rule would be if it has an Objective that needs to be completed before it can be inserted, and that objective hasn't been completed yet. In practice, a socket will *either* have reusable plugs *or* it will allow for plugs in your inventory to be inserted. See DestinyInventoryItemDefinition.socket for more info. KEY = The INDEX into the item's list of sockets. VALUE = The set of plugs for that socket. If a socket doesn't have any reusable plugs defined at the item scope, there will be no entry for that socket. |

#### Destiny.Components.Items.DestinyItemPlugObjectivesComponentDepends on Component "ItemPlugObjectives"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `objectivesPerPlug` | Mapping&lt;uint32, array&gt; → DestinyInventoryItemDefinition | This set of data is keyed by the Item Hash (DestinyInventoryItemDefinition) of the plug whose objectives are being returned, with the value being the list of those objectives. What if two plugs with the same hash are returned for an item, you ask? Good question! They share the same item-scoped state, and as such would have identical objective state as a result. How's that for convenient. Sometimes, Plugs may have objectives: generally, these are used for flavor and display purposes. For instance, a Plug might be tracking the number of PVP kills you have made. It will use the parent item's data about that tracking status to determine what to show, and will generally show it using the DestinyObjectiveDefinition's progressDescription property. Refer to the plug's itemHash and objective property for more information if you would like to display even more data. |

#### Destiny.Entities.Items.DestinyItemTalentGridComponentDepends on Component "ItemTalentGrids"

**Type:** object

Well, we're here in Destiny 2, and Talent Grids are unfortunately still around. The good news is that they're pretty much only being used for certain base information on items and for Builds/Subclasses. The bad news is that they still suck. If you really want this information, grab this component. An important note is that talent grids are defined as such: A Grid has 1:M Nodes, which has 1:M Steps. Any given node can only have a single step active at one time, which represents the actual visual contents and effects of the Node (for instance, if you see a "Super Cool Bonus" node, the actual icon and text for the node is coming from the current Step of that node). Nodes can be grouped into exclusivity sets *and* as of D2, exclusivity groups (which are collections of exclusivity sets that affect each other). See DestinyTalentGridDefinition for more information. Brace yourself, the water's cold out there in the deep end.

| Property | Type | Description |
| --- | --- | --- |
| `talentGridHash` | uint32 → DestinyTalentGridDefinition | Most items don't have useful talent grids anymore, but Builds in particular still do. You can use this hash to lookup the DestinyTalentGridDefinition attached to this item, which will be crucial for understanding the node values on the item. |
| `nodes` | array&lt;Destiny.DestinyTalentNode&gt; | Detailed information about the individual nodes in the talent grid. A node represents a single visual "pip" in the talent grid or Build detail view, though each node may have multiple "steps" which indicate the actual bonuses and visual representation of that node. |
| `isGridComplete` | boolean | Indicates whether the talent grid on this item is completed, and thus whether it should have a gold border around it. Only will be true if the item actually *has* a talent grid, and only then if it is completed (i.e. every exclusive set has an activated node, and every non-exclusive set node has been activated) |
| `gridProgression` | Destiny.DestinyProgression | If the item has a progression, it will be detailed here. A progression means that the item can gain experience. Thresholds of experience are what determines whether and when a talent node can be activated. |

#### Destiny.DestinyTalentNode

**Type:** object

I see you've come to find out more about Talent Nodes. I'm so sorry. Talent Nodes are the conceptual, visual nodes that appear on Talent Grids. Talent Grids, in Destiny 1, were found on almost every instanced item: they had Nodes that could be activated to change the properties of the item. In Destiny 2, Talent Grids only exist for Builds/Subclasses, and while the basic concept is the same (Nodes can be activated once you've gained sufficient Experience on the Item, and provide effects), there are some new concepts from Destiny 1. Examine DestinyTalentGridDefinition and its subordinates for more information. This is the "Live" information for the current status of a Talent Node on a specific item. Talent Nodes have many Steps, but only one can be active at any one time: and it is the Step that determines both the visual and the game state-changing properties that the Node provides. Examine this and DestinyTalentNodeStepDefinition carefully. *IMPORTANT NOTE* Talent Nodes are, unfortunately, Content Version DEPENDENT. Though they refer to hashes for Nodes and Steps, those hashes are not guaranteed to be immutable across content versions. This is a source of great exasperation for me, but as a result anyone using Talent Grid data must ensure that the content version of their static content matches that of the server responses before showing or making decisions based on talent grid data.

| Property | Type | Description |
| --- | --- | --- |
| `nodeIndex` | int32 | The index of the Talent Node being referred to (an index into DestinyTalentGridDefinition.nodes[]). CONTENT VERSION DEPENDENT. |
| `nodeHash` | uint32 | The hash of the Talent Node being referred to (in DestinyTalentGridDefinition.nodes). Deceptively CONTENT VERSION DEPENDENT. We have no guarantee of the hash's immutability between content versions. |
| `state` | int32 | An DestinyTalentNodeState enum value indicating the node's state: whether it can be activated or swapped, and why not if neither can be performed. |
| `isActivated` | boolean | If true, the node is activated: it's current step then provides its benefits. |
| `stepIndex` | int32 | The currently relevant Step for the node. It is this step that has rendering data for the node and the benefits that are provided if the node is activated. (the actual rules for benefits provided are extremely complicated in theory, but with how Talent Grids are being used in Destiny 2 you don't have to worry about a lot of those old Destiny 1 rules.) This is an index into: DestinyTalentGridDefinition.nodes[nodeIndex].steps[stepIndex] |
| `materialsToUpgrade` | array&lt;Destiny.Definitions.DestinyMaterialRequirement&gt; | If the node has material requirements to be activated, this is the list of those requirements. |
| `activationGridLevel` | int32 | The progression level required on the Talent Grid in order to be able to activate this talent node. Talent Grids have their own Progression - similar to Character Level, but in this case it is experience related to the item itself. |
| `progressPercent` | float | If you want to show a progress bar or circle for how close this talent node is to being activate- able, this is the percentage to show. It follows the node's underlying rules about when the progress bar should first show up, and when it should be filled. |
| `hidden` | boolean | Whether or not the talent node is actually visible in the game's UI. Whether you want to show it in your own UI is up to you! I'm not gonna tell you who to sock it to. |
| `nodeStatsBlock` | Destiny.DestinyTalentNodeStatBlock | This property has some history. A talent grid can provide stats on both the item it's related to and the character equipping the item. This returns data about those stat bonuses. |

#### Destiny.DestinyTalentNodeStateEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Invalid` | 0 | — |
| `CanUpgrade` | 1 | — |
| `NoPoints` | 2 | — |
| `NoPrerequisites` | 3 | — |
| `NoSteps` | 4 | — |
| `NoUnlock` | 5 | — |
| `NoMaterial` | 6 | — |
| `NoGridLevel` | 7 | — |
| `SwappingLocked` | 8 | — |
| `MustSwap` | 9 | — |
| `Complete` | 10 | — |
| `Unknown` | 11 | — |
| `CreationOnly` | 12 | — |
| `Hidden` | 13 | — |

#### Destiny.DestinyTalentNodeStatBlock

**Type:** object

This property has some history. A talent grid can provide stats on both the item it's related to and the character equipping the item. This returns data about those stat bonuses.

| Property | Type | Description |
| --- | --- | --- |
| `currentStepStats` | array&lt;Destiny.DestinyStat&gt; | The stat benefits conferred when this talent node is activated for the current Step that is active on the node. |
| `nextStepStats` | array&lt;Destiny.DestinyStat&gt; | This is a holdover from the old days of Destiny 1, when a node could be activated multiple times, conferring multiple steps worth of benefits: you would use this property to show what activating the "next" step on the node would provide vs. what the current step is providing. While Nodes are currently not being used this way, the underlying system for this functionality still exists. I hesitate to remove this property while the ability for designers to make such a talent grid still exists. Whether you want to show it is up to you. |

#### Destiny.Components.Items.DestinyItemPlugComponentDepends on Component "ItemPlugStates"

**Type:** object

Plugs are non-instanced items that can provide Stat and Perk benefits when socketed into an instanced item. Items have Sockets, and Plugs are inserted into Sockets. This component finds all items that are considered "Plugs" in your inventory, and return information about the plug aside from any specific Socket into which it could be inserted.

| Property | Type | Description |
| --- | --- | --- |
| `plugObjectives` | array&lt;Destiny.Quests.DestinyObjectiveProgress&gt; | Sometimes, Plugs may have objectives: these are often used for flavor and display purposes, but they can be used for any arbitrary purpose (both fortunately and unfortunately). Recently (with Season 2) they were expanded in use to be used as the "gating" for whether the plug can be inserted at all. For instance, a Plug might be tracking the number of PVP kills you have made. It will use the parent item's data about that tracking status to determine what to show, and will generally show it using the DestinyObjectiveDefinition's progressDescription property. Refer to the plug's itemHash and objective property for more information if you would like to display even more data. |
| `plugItemHash` | uint32 → DestinyInventoryItemDefinition | The hash identifier of the DestinyInventoryItemDefinition that represents this plug. |
| `canInsert` | boolean | If true, this plug has met all of its insertion requirements. Big if true. |
| `enabled` | boolean | If true, this plug will provide its benefits while inserted. |
| `insertFailIndexes` | array&lt;int32&gt; | If the plug cannot be inserted for some reason, this will have the indexes into the plug item definition's plug.insertionRules property, so you can show the reasons why it can't be inserted. This list will be empty if the plug can be inserted. |
| `enableFailIndexes` | array&lt;int32&gt; | If a plug is not enabled, this will be populated with indexes into the plug item definition's plug.enabledRules property, so that you can show the reasons why it is not enabled. This list will be empty if the plug is enabled. |
| `stackSize` | int32? | If available, this is the stack size to display for the socket plug item. |
| `maxStackSize` | int32? | If available, this is the maximum stack size to display for the socket plug item. |

#### Destiny.Components.Inventory.DestinyCurrenciesComponentDepends on Component "CurrencyLookups"

**Type:** object

This component provides a quick lookup of every item the requested character has and how much of that item they have. Requesting this component will allow you to circumvent manually putting together the list of which currencies you have for the purpose of testing currency requirements on an item being purchased, or operations that have costs. You *could* figure this out yourself by doing a GetCharacter or GetProfile request and forming your own lookup table, but that is inconvenient enough that this feels like a worthwhile (and optional) redundancy. Don't bother requesting it if you have already created your own lookup from prior GetCharacter/GetProfile calls.

| Property | Type | Description |
| --- | --- | --- |
| `itemQuantities` | Mapping&lt;uint32, int32&gt; → DestinyInventoryItemDefinition | A dictionary - keyed by the item's hash identifier (DestinyInventoryItemDefinition), and whose value is the amount of that item you have across all available inventory buckets for purchasing. This allows you to see whether the requesting character can afford any given purchase/action without having to re-create this list itself. |
| `materialRequirementSetStates` | Mapping&lt;uint32, Destiny.Components.Inventory.DestinyMaterialRequirementSetState&gt; | A map of material requirement hashes and their status information. |

#### Destiny.Components.Inventory.DestinyMaterialRequirementSetState

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `materialRequirementSetHash` | uint32 → DestinyMaterialRequirementSetDefinition | The hash identifier of the material requirement set. Use it to look up the DestinyMaterialRequirementSetDefinition. |
| `materialRequirementStates` | array&lt;Destiny.Components.Inventory.DestinyMaterialRequirementState&gt; | The dynamic state values for individual material requirements. |

#### Destiny.Components.Inventory.DestinyMaterialRequirementState

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `itemHash` | uint32 | The hash identifier of the material required. Use it to look up the material's DestinyInventoryItemDefinition. |
| `count` | int32 | The amount of the material required. |
| `stackSize` | int32 | A value for the amount of a (possibly virtual) material on some scope. For example: Dawning cookie baking material requirements. |

#### Destiny.Responses.DestinyCharacterResponse

**Type:** object

The response contract for GetDestinyCharacter, with components that can be returned for character and item-level data.

| Property | Type | Description |
| --- | --- | --- |
| `inventory` | SingleComponentResponseOfDestinyInventoryComponent | The character-level non-equipped inventory items. COMPONENT TYPE: CharacterInventories |
| `character` | SingleComponentResponseOfDestinyCharacterComponent | Base information about the character in question. COMPONENT TYPE: Characters |
| `progressions` | SingleComponentResponseOfDestinyCharacterProgressionComponent | Character progression data, including Milestones. COMPONENT TYPE: CharacterProgressions |
| `renderData` | SingleComponentResponseOfDestinyCharacterRenderComponent | Character rendering data - a minimal set of information about equipment and dyes used for rendering. COMPONENT TYPE: CharacterRenderData |
| `activities` | SingleComponentResponseOfDestinyCharacterActivitiesComponent | Activity data - info about current activities available to the player. COMPONENT TYPE: CharacterActivities |
| `equipment` | SingleComponentResponseOfDestinyInventoryComponent | Equipped items on the character. COMPONENT TYPE: CharacterEquipment |
| `loadouts` | SingleComponentResponseOfDestinyLoadoutsComponent | The loadouts available to the character. COMPONENT TYPE: CharacterLoadouts |
| `kiosks` | SingleComponentResponseOfDestinyKiosksComponent | Items available from Kiosks that are available to this specific character. COMPONENT TYPE: Kiosks |
| `plugSets` | SingleComponentResponseOfDestinyPlugSetsComponent | When sockets refer to reusable Plug Sets (see DestinyPlugSetDefinition for more info), this is the set of plugs and their states that are scoped to this character. This comes back with ItemSockets, as it is needed for a complete picture of the sockets on requested items. COMPONENT TYPE: ItemSockets |
| `presentationNodes` | SingleComponentResponseOfDestinyPresentationNodesComponent | COMPONENT TYPE: PresentationNodes |
| `records` | SingleComponentResponseOfDestinyCharacterRecordsComponent | COMPONENT TYPE: Records |
| `collectibles` | SingleComponentResponseOfDestinyCollectiblesComponent | COMPONENT TYPE: Collectibles |
| `itemComponents` | DestinyItemComponentSetOfint64 | The set of components belonging to the player's instanced items. COMPONENT TYPE: [See inside the DestinyItemComponentSet contract for component types.] |
| `uninstancedItemComponents` | DestinyBaseItemComponentSetOfuint32 | The set of components belonging to the player's UNinstanced items. Because apparently now those too can have information relevant to the character's state. COMPONENT TYPE: [See inside the DestinyItemComponentSet contract for component types.] |
| `currencyLookups` | SingleComponentResponseOfDestinyCurrenciesComponent | A "lookup" convenience component that can be used to quickly check if the character has access to items that can be used for purchasing. COMPONENT TYPE: CurrencyLookups |

#### Destiny.Responses.DestinyItemResponse

**Type:** object

The response object for retrieving an individual instanced item. None of these components are relevant for an item that doesn't have an "itemInstanceId": for those, get your information from the DestinyInventoryDefinition.

| Property | Type | Description |
| --- | --- | --- |
| `characterId` | int64? | If the item is on a character, this will return the ID of the character that is holding the item. |
| `item` | SingleComponentResponseOfDestinyItemComponent | Common data for the item relevant to its non-instanced properties. COMPONENT TYPE: ItemCommonData |
| `instance` | SingleComponentResponseOfDestinyItemInstanceComponent | Basic instance data for the item. COMPONENT TYPE: ItemInstances |
| `objectives` | SingleComponentResponseOfDestinyItemObjectivesComponent | Information specifically about the item's objectives. COMPONENT TYPE: ItemObjectives |
| `perks` | SingleComponentResponseOfDestinyItemPerksComponent | Information specifically about the perks currently active on the item. COMPONENT TYPE: ItemPerks |
| `renderData` | SingleComponentResponseOfDestinyItemRenderComponent | Information about how to render the item in 3D. COMPONENT TYPE: ItemRenderData |
| `stats` | SingleComponentResponseOfDestinyItemStatsComponent | Information about the computed stats of the item: power, defense, etc... COMPONENT TYPE: ItemStats |
| `talentGrid` | SingleComponentResponseOfDestinyItemTalentGridComponent | Information about the talent grid attached to the item. Talent nodes can provide a variety of benefits and abilities, and in Destiny 2 are used almost exclusively for the character's "Builds". COMPONENT TYPE: ItemTalentGrids |
| `sockets` | SingleComponentResponseOfDestinyItemSocketsComponent | Information about the sockets of the item: which are currently active, what potential sockets you could have and the stats/abilities/perks you can gain from them. COMPONENT TYPE: ItemSockets |
| `reusablePlugs` | SingleComponentResponseOfDestinyItemReusablePlugsComponent | Information about the Reusable Plugs for sockets on an item. These are plugs that you can insert into the given socket regardless of if you actually own an instance of that plug: they are logic- driven plugs rather than inventory-driven. These may need to be combined with Plug Set component data to get a full picture of available plugs on a given socket. COMPONENT TYPE: ItemReusablePlugs |
| `plugObjectives` | SingleComponentResponseOfDestinyItemPlugObjectivesComponent | Information about objectives on Plugs for a given item. See the component's documentation for more info. COMPONENT TYPE: ItemPlugObjectives |

#### Destiny.DestinyVendorFilterEnumeration

**Enum** (`int32`)

Indicates the type of filter to apply to Vendor results.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `ApiPurchasable` | 1 | — |

#### Destiny.Responses.DestinyVendorsResponse

**Type:** object

A response containing all of the components for all requested vendors.

| Property | Type | Description |
| --- | --- | --- |
| `vendorGroups` | SingleComponentResponseOfDestinyVendorGroupComponent | For Vendors being returned, this will give you the information you need to group them and order them in the same way that the Bungie Companion app performs grouping. It will automatically be returned if you request the Vendors component. COMPONENT TYPE: Vendors |
| `vendors` | DictionaryComponentResponseOfuint32AndDestinyVendorComponent | The base properties of the vendor. These are keyed by the Vendor Hash, so you will get one Vendor Component per vendor returned. COMPONENT TYPE: Vendors |
| `categories` | DictionaryComponentResponseOfuint32AndDestinyVendorCategoriesComponent | Categories that the vendor has available, and references to the sales therein. These are keyed by the Vendor Hash, so you will get one Categories Component per vendor returned. COMPONENT TYPE: VendorCategories |
| `sales` | DictionaryComponentResponseOfuint32AndPersonalDestinyVendorSaleItemSetComponent | Sales, keyed by the vendorItemIndex of the item being sold. These are keyed by the Vendor Hash, so you will get one Sale Item Set Component per vendor returned. Note that within the Sale Item Set component, the sales are themselves keyed by the vendorSaleIndex, so you can relate it to the current sale item definition within the Vendor's definition. COMPONENT TYPE: VendorSales |
| `itemComponents` | Mapping&lt;uint32, DestinyVendorItemComponentSetOfint32&gt; | The set of item detail components, one set of item components per Vendor. These are keyed by the Vendor Hash, so you will get one Item Component Set per vendor returned. The components contained inside are themselves keyed by the vendorSaleIndex, and will have whatever item-level components you requested (Sockets, Stats, Instance data etc...) per item being sold by the vendor. |
| `currencyLookups` | SingleComponentResponseOfDestinyCurrenciesComponent | A "lookup" convenience component that can be used to quickly check if the character has access to items that can be used for purchasing. COMPONENT TYPE: CurrencyLookups |
| `stringVariables` | SingleComponentResponseOfDestinyStringVariablesComponent | A map of string variable values by hash for this character context. COMPONENT TYPE: StringVariables |

#### Destiny.Components.Vendors.DestinyVendorGroupComponentDepends on Component "Vendors"

**Type:** object

This component returns references to all of the Vendors in the response, grouped by categorizations that Bungie has deemed to be interesting, in the order in which both the groups and the vendors within that group should be rendered.

| Property | Type | Description |
| --- | --- | --- |
| `groups` | array&lt;Destiny.Components.Vendors.DestinyVendorGroup&gt; | The ordered list of groups being returned. |

#### Destiny.Components.Vendors.DestinyVendorGroup

**Type:** object

Represents a specific group of vendors that can be rendered in the recommended order. How do we figure out this order? It's a long story, and will likely get more complicated over time.

| Property | Type | Description |
| --- | --- | --- |
| `vendorGroupHash` | uint32 → DestinyVendorGroupDefinition | — |
| `vendorHashes` | array&lt;uint32&gt; → DestinyVendorDefinition | The ordered list of vendors within a particular group. |

#### Destiny.Components.Vendors.DestinyVendorBaseComponentDepends on Component "Vendors"

**Type:** object

This component contains essential/summary information about the vendor.

| Property | Type | Description |
| --- | --- | --- |
| `vendorHash` | uint32 → DestinyVendorDefinition | The unique identifier for the vendor. Use it to look up their DestinyVendorDefinition. |
| `nextRefreshDate` | date-time | The date when this vendor's inventory will next rotate/refresh. Note that this is distinct from the date ranges that the vendor is visible/available in-game: this field indicates the specific time when the vendor's available items refresh and rotate, regardless of whether the vendor is actually available at that time. Unfortunately, these two values may be (and are, for the case of important vendors like Xur) different. Issue <https://github.com/Bungie-net/api/issues/353> is tracking a fix to start providing visibility date ranges where possible in addition to this refresh date, so that all important dates for vendors are available for use. |
| `enabled` | boolean | If True, the Vendor is currently accessible. If False, they may not actually be visible in the world at the moment. |

#### Destiny.Entities.Vendors.DestinyVendorComponentDepends on Component "Vendors"

**Type:** object

This component contains essential/summary information about the vendor.

| Property | Type | Description |
| --- | --- | --- |
| `canPurchase` | boolean | If True, you can purchase from the Vendor. |
| `progression` | Destiny.DestinyProgression | If the Vendor has a related Reputation, this is the Progression data that represents the character's Reputation level with this Vendor. |
| `vendorLocationIndex` | int32 | An index into the vendor definition's "locations" property array, indicating which location they are at currently. If -1, then the vendor has no known location (and you may choose not to show them in your UI as a result. I mean, it's your bag honey) |
| `seasonalRank` | int32? | If this vendor has a seasonal rank, this will be the calculated value of that rank. How nice is that? I mean, that's pretty sweeet. It's a whole 32 bit integer. |
| `vendorHash` | uint32 → DestinyVendorDefinition | The unique identifier for the vendor. Use it to look up their DestinyVendorDefinition. |
| `nextRefreshDate` | date-time | The date when this vendor's inventory will next rotate/refresh. Note that this is distinct from the date ranges that the vendor is visible/available in-game: this field indicates the specific time when the vendor's available items refresh and rotate, regardless of whether the vendor is actually available at that time. Unfortunately, these two values may be (and are, for the case of important vendors like Xur) different. Issue <https://github.com/Bungie-net/api/issues/353> is tracking a fix to start providing visibility date ranges where possible in addition to this refresh date, so that all important dates for vendors are available for use. |
| `enabled` | boolean | If True, the Vendor is currently accessible. If False, they may not actually be visible in the world at the moment. |

#### Destiny.Entities.Vendors.DestinyVendorCategoriesComponentDepends on Component "VendorCategories"

**Type:** object

A vendor can have many categories of items that they sell. This component will return the category information for available items, as well as the index into those items in the user's sale item list. Note that, since both the category and items are indexes, this data is Content Version dependent. Be sure to check that your content is up to date before using this data. This is an unfortunate, but permanent, limitation of Vendor data.

| Property | Type | Description |
| --- | --- | --- |
| `categories` | array&lt;Destiny.Entities.Vendors.DestinyVendorCategory&gt; | The list of categories for items that the vendor sells, in rendering order. These categories each point to a "display category" in the displayCategories property of the DestinyVendorDefinition, as opposed to the other categories. |

#### Destiny.Entities.Vendors.DestinyVendorCategory

**Type:** object

Information about the category and items currently sold in that category.

| Property | Type | Description |
| --- | --- | --- |
| `displayCategoryIndex` | int32 | An index into the DestinyVendorDefinition.displayCategories property, so you can grab the display data for this category. |
| `itemIndexes` | array&lt;int32&gt; | An ordered list of indexes into items being sold in this category (DestinyVendorDefinition.itemList) which will contain more information about the items being sold themselves. Can also be used to index into DestinyVendorSaleItemComponent data, if you asked for that data to be returned. |

#### Destiny.Components.Vendors.DestinyVendorSaleItemBaseComponentDepends on Component "VendorSales"

**Type:** object

The base class for Vendor Sale Item data. Has a bunch of character-agnostic state about the item being sold. Note that if you want instance, stats, etc... data for the item, you'll have to request additional components such as ItemInstances, ItemPerks etc... and acquire them from the DestinyVendorResponse's "items" property.

| Property | Type | Description |
| --- | --- | --- |
| `vendorItemIndex` | int32 | The index into the DestinyVendorDefinition.itemList property. Note that this means Vendor data *is* Content Version dependent: make sure you have the latest content before you use Vendor data, or these indexes may mismatch. Most systems avoid this problem, but Vendors is one area where we are unable to reasonably avoid content dependency at the moment. |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | The hash of the item being sold, as a quick shortcut for looking up the DestinyInventoryItemDefinition of the sale item. |
| `overrideStyleItemHash` | uint32 → DestinyInventoryItemDefinition? | If populated, this is the hash of the item whose icon (and other secondary styles, but *not* the human readable strings) should override whatever icons/styles are on the item being sold. If you don't do this, certain items whose styles are being overridden by socketed items - such as the "Recycle Shader" item - would show whatever their default icon/style is, and it wouldn't be pretty or look accurate. |
| `quantity` | int32 | How much of the item you'll be getting. |
| `costs` | array&lt;Destiny.DestinyItemQuantity&gt; | A summary of the current costs of the item. |
| `overrideNextRefreshDate` | date-time? | If this item has its own custom date where it may be removed from the Vendor's rotation, this is that date. Note that there's not actually any guarantee that it will go away: it could be chosen again and end up still being in the Vendor's sale items! But this is the next date where that test will occur, and is also the date that the game shows for availability on things like Bounties being sold. So it's the best we can give. |
| `apiPurchasable` | boolean? | If true, this item can be purchased through the Bungie.net API. |

#### Destiny.Entities.Vendors.DestinyVendorSaleItemComponentDepends on Component "VendorSales"

**Type:** object

Request this component if you want the details about an item being sold in relation to the character making the request: whether the character can buy it, whether they can afford it, and other data related to purchasing the item. Note that if you want instance, stats, etc... data for the item, you'll have to request additional components such as ItemInstances, ItemPerks etc... and acquire them from the DestinyVendorResponse's "items" property.

| Property | Type | Description |
| --- | --- | --- |
| `saleStatus` | int32 | A flag indicating whether the requesting character can buy the item, and if not the reasons why the character can't buy it. |
| `requiredUnlocks` | array&lt;uint32&gt; → DestinyUnlockDefinition | If you can't buy the item due to a complex character state, these will be hashes for DestinyUnlockDefinitions that you can check to see messages regarding the failure (if the unlocks have human readable information: it is not guaranteed that Unlocks will have human readable strings, and your application will have to handle that) Prefer using failureIndexes instead. These are provided for informational purposes, but have largely been supplanted by failureIndexes. |
| `unlockStatuses` | array&lt;Destiny.DestinyUnlockStatus&gt; | If any complex unlock states are checked in determining purchasability, these will be returned here along with the status of the unlock check. Prefer using failureIndexes instead. These are provided for informational purposes, but have largely been supplanted by failureIndexes. |
| `failureIndexes` | array&lt;int32&gt; | Indexes in to the "failureStrings" lookup table in DestinyVendorDefinition for the given Vendor. Gives some more reliable failure information for why you can't purchase an item. It is preferred to use these over requiredUnlocks and unlockStatuses: the latter are provided mostly in case someone can do something interesting with it that I didn't anticipate. |
| `augments` | int32 | A flags enumeration value representing the current state of any "state modifiers" on the item being sold. These are meant to correspond with some sort of visual indicator as to the augmentation: for instance, if an item is on sale or if you already own the item in question. Determining how you want to represent these in your own app (or if you even want to) is an exercise left for the reader. |
| `itemValueVisibility` | array&lt;boolean&gt; | If available, a list that describes which item values (rewards) should be shown (true) or hidden (false). |
| `vendorItemIndex` | int32 | The index into the DestinyVendorDefinition.itemList property. Note that this means Vendor data *is* Content Version dependent: make sure you have the latest content before you use Vendor data, or these indexes may mismatch. Most systems avoid this problem, but Vendors is one area where we are unable to reasonably avoid content dependency at the moment. |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | The hash of the item being sold, as a quick shortcut for looking up the DestinyInventoryItemDefinition of the sale item. |
| `overrideStyleItemHash` | uint32 → DestinyInventoryItemDefinition? | If populated, this is the hash of the item whose icon (and other secondary styles, but *not* the human readable strings) should override whatever icons/styles are on the item being sold. If you don't do this, certain items whose styles are being overridden by socketed items - such as the "Recycle Shader" item - would show whatever their default icon/style is, and it wouldn't be pretty or look accurate. |
| `quantity` | int32 | How much of the item you'll be getting. |
| `costs` | array&lt;Destiny.DestinyItemQuantity&gt; | A summary of the current costs of the item. |
| `overrideNextRefreshDate` | date-time? | If this item has its own custom date where it may be removed from the Vendor's rotation, this is that date. Note that there's not actually any guarantee that it will go away: it could be chosen again and end up still being in the Vendor's sale items! But this is the next date where that test will occur, and is also the date that the game shows for availability on things like Bounties being sold. So it's the best we can give. |
| `apiPurchasable` | boolean? | If true, this item can be purchased through the Bungie.net API. |

#### Destiny.VendorItemStatusEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Success` | 0 | — |
| `NoInventorySpace` | 1 | — |
| `NoFunds` | 2 | — |
| `NoProgression` | 4 | — |
| `NoUnlock` | 8 | — |
| `NoQuantity` | 16 | — |
| `OutsidePurchaseWindow` | 32 | — |
| `NotAvailable` | 64 | — |
| `UniquenessViolation` | 128 | — |
| `UnknownError` | 256 | — |
| `AlreadySelling` | 512 | — |
| `Unsellable` | 1024 | — |
| `SellingInhibited` | 2048 | — |
| `AlreadyOwned` | 4096 | DEPRECATED - Owned items use the NoUnlock state and a failure string indicating the proper display state. |
| `DisplayOnly` | 8192 | — |

#### Destiny.DestinyUnlockStatus

**Type:** object

Indicates the status of an "Unlock Flag" on a Character or Profile. These are individual bits of state that can be either set or not set, and sometimes provide interesting human-readable information in their related DestinyUnlockDefinition.

| Property | Type | Description |
| --- | --- | --- |
| `unlockHash` | uint32 → DestinyUnlockDefinition | The hash identifier for the Unlock Flag. Use to lookup DestinyUnlockDefinition for static data. Not all unlocks have human readable data - in fact, most don't. But when they do, it can be very useful to show. Even if they don't have human readable data, you might be able to infer the meaning of an unlock flag with a bit of experimentation... |
| `isSet` | boolean | Whether the unlock flag is set. |

#### Destiny.DestinyVendorItemStateEnumeration

**Enum** (`int32`)

The possible states of Destiny Profile Records. IMPORTANT: Any given item can theoretically have many of these states simultaneously: as a result, this was altered to be a flags enumeration/bitmask for v3.2.0.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | There are no augments on the item. |
| `Incomplete` | 1 | Deprecated forever (probably). There was a time when Records were going to be implemented through Vendors, and this field was relevant. Now they're implemented through Presentation Nodes, and this field doesn't matter anymore. |
| `RewardAvailable` | 2 | Deprecated forever (probably). See the description of the "Incomplete" value for the juicy scoop. |
| `Complete` | 4 | Deprecated forever (probably). See the description of the "Incomplete" value for the juicy scoop. |
| `New` | 8 | This item is considered to be "newly available", and should have some UI showing how shiny it is. |
| `Featured` | 16 | This item is being "featured", and should be shiny in a different way from items that are merely new. |
| `Ending` | 32 | This item is only available for a limited time, and that time is approaching. |
| `OnSale` | 64 | This item is "on sale". Get it while it's hot. |
| `Owned` | 128 | This item is already owned. |
| `WideView` | 256 | This item should be shown with a "wide view" instead of normal icon view. |
| `NexusAttention` | 512 | This indicates that you should show some kind of attention-requesting indicator on the item, in a similar manner to items in the nexus that have such notifications. |
| `SetDiscount` | 1024 | This indicates that the item has some sort of a 'set' discount. |
| `PriceDrop` | 2048 | This indicates that the item has a price drop. |
| `DailyOffer` | 4096 | This indicates that the item is a daily offer. |
| `Charity` | 8192 | This indicates that the item is for charity. |
| `SeasonalRewardExpiration` | 16384 | This indicates that the item has a seasonal reward expiration. |
| `BestDeal` | 32768 | This indicates that the sale item is the best deal among different choices. |
| `Popular` | 65536 | This indicates that the sale item is popular. |
| `Free` | 131072 | This indicates that the sale item is free. |
| `Locked` | 262144 | This indicates that the sale item is locked. |
| `Paracausal` | 524288 | This indicates that the sale item is paracausal. |
| `Cryptarch` | 1048576 | — |
| `ArtifactPerkOwned` | 2097152 | — |
| `Savings` | 4194304 | — |
| `Ineligible` | 8388608 | — |
| `ArtifactPerkBoosted` | 16777216 | — |
| `SeasonalArchiveFree` | 33554432 | — |

#### Destiny.Responses.PersonalDestinyVendorSaleItemSetComponentDepends on Component "VendorSales"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `saleItems` | Mapping&lt;int32, Destiny.Entities.Vendors.DestinyVendorSaleItemComponent&gt; | — |

#### Destiny.Responses.DestinyVendorResponse

**Type:** object

A response containing all of the components for a vendor.

| Property | Type | Description |
| --- | --- | --- |
| `vendor` | SingleComponentResponseOfDestinyVendorComponent | The base properties of the vendor. COMPONENT TYPE: Vendors |
| `categories` | SingleComponentResponseOfDestinyVendorCategoriesComponent | Categories that the vendor has available, and references to the sales therein. COMPONENT TYPE: VendorCategories |
| `sales` | DictionaryComponentResponseOfint32AndDestinyVendorSaleItemComponent | Sales, keyed by the vendorItemIndex of the item being sold. COMPONENT TYPE: VendorSales |
| `itemComponents` | DestinyVendorItemComponentSetOfint32 | Item components, keyed by the vendorItemIndex of the active sale items. COMPONENT TYPE: [See inside the DestinyVendorItemComponentSet contract for component types.] |
| `currencyLookups` | SingleComponentResponseOfDestinyCurrenciesComponent | A "lookup" convenience component that can be used to quickly check if the character has access to items that can be used for purchasing. COMPONENT TYPE: CurrencyLookups |
| `stringVariables` | SingleComponentResponseOfDestinyStringVariablesComponent | A map of string variable values by hash for this character context. COMPONENT TYPE: StringVariables |

#### Destiny.Responses.DestinyPublicVendorsResponse

**Type:** object

A response containing all valid components for the public Vendors endpoint. It is a decisively smaller subset of data compared to what we can get when we know the specific user making the request. If you want any of the other data - item details, whether or not you can buy it, etc... you'll have to call in the context of a character. I know, sad but true.

| Property | Type | Description |
| --- | --- | --- |
| `vendorGroups` | SingleComponentResponseOfDestinyVendorGroupComponent | For Vendors being returned, this will give you the information you need to group them and order them in the same way that the Bungie Companion app performs grouping. It will automatically be returned if you request the Vendors component. COMPONENT TYPE: Vendors |
| `vendors` | DictionaryComponentResponseOfuint32AndDestinyPublicVendorComponent | The base properties of the vendor. These are keyed by the Vendor Hash, so you will get one Vendor Component per vendor returned. COMPONENT TYPE: Vendors |
| `categories` | DictionaryComponentResponseOfuint32AndDestinyVendorCategoriesComponent | Categories that the vendor has available, and references to the sales therein. These are keyed by the Vendor Hash, so you will get one Categories Component per vendor returned. COMPONENT TYPE: VendorCategories |
| `sales` | DictionaryComponentResponseOfuint32AndPublicDestinyVendorSaleItemSetComponent | Sales, keyed by the vendorItemIndex of the item being sold. These are keyed by the Vendor Hash, so you will get one Sale Item Set Component per vendor returned. Note that within the Sale Item Set component, the sales are themselves keyed by the vendorSaleIndex, so you can relate it to the corrent sale item definition within the Vendor's definition. COMPONENT TYPE: VendorSales |
| `stringVariables` | SingleComponentResponseOfDestinyStringVariablesComponent | A set of string variable values by hash for a public vendors context. COMPONENT TYPE: StringVariables |

#### Destiny.Components.Vendors.DestinyPublicVendorComponentDepends on Component "Vendors"

**Type:** object

This component contains essential/summary information about the vendor from the perspective of a character-agnostic view.

| Property | Type | Description |
| --- | --- | --- |
| `vendorHash` | uint32 → DestinyVendorDefinition | The unique identifier for the vendor. Use it to look up their DestinyVendorDefinition. |
| `nextRefreshDate` | date-time | The date when this vendor's inventory will next rotate/refresh. Note that this is distinct from the date ranges that the vendor is visible/available in-game: this field indicates the specific time when the vendor's available items refresh and rotate, regardless of whether the vendor is actually available at that time. Unfortunately, these two values may be (and are, for the case of important vendors like Xur) different. Issue <https://github.com/Bungie-net/api/issues/353> is tracking a fix to start providing visibility date ranges where possible in addition to this refresh date, so that all important dates for vendors are available for use. |
| `enabled` | boolean | If True, the Vendor is currently accessible. If False, they may not actually be visible in the world at the moment. |

#### Destiny.Components.Vendors.DestinyPublicVendorSaleItemComponentDepends on Component "VendorSales"

**Type:** object

Has character-agnostic information about an item being sold by a vendor. Note that if you want instance, stats, etc... data for the item, you'll have to request additional components such as ItemInstances, ItemPerks etc... and acquire them from the DestinyVendorResponse's "items" property. For most of these, however, you'll have to ask for it in context of a specific character.

| Property | Type | Description |
| --- | --- | --- |
| `vendorItemIndex` | int32 | The index into the DestinyVendorDefinition.itemList property. Note that this means Vendor data *is* Content Version dependent: make sure you have the latest content before you use Vendor data, or these indexes may mismatch. Most systems avoid this problem, but Vendors is one area where we are unable to reasonably avoid content dependency at the moment. |
| `itemHash` | uint32 → DestinyInventoryItemDefinition | The hash of the item being sold, as a quick shortcut for looking up the DestinyInventoryItemDefinition of the sale item. |
| `overrideStyleItemHash` | uint32 → DestinyInventoryItemDefinition? | If populated, this is the hash of the item whose icon (and other secondary styles, but *not* the human readable strings) should override whatever icons/styles are on the item being sold. If you don't do this, certain items whose styles are being overridden by socketed items - such as the "Recycle Shader" item - would show whatever their default icon/style is, and it wouldn't be pretty or look accurate. |
| `quantity` | int32 | How much of the item you'll be getting. |
| `costs` | array&lt;Destiny.DestinyItemQuantity&gt; | A summary of the current costs of the item. |
| `overrideNextRefreshDate` | date-time? | If this item has its own custom date where it may be removed from the Vendor's rotation, this is that date. Note that there's not actually any guarantee that it will go away: it could be chosen again and end up still being in the Vendor's sale items! But this is the next date where that test will occur, and is also the date that the game shows for availability on things like Bounties being sold. So it's the best we can give. |
| `apiPurchasable` | boolean? | If true, this item can be purchased through the Bungie.net API. |

#### Destiny.Responses.PublicDestinyVendorSaleItemSetComponentDepends on Component "VendorSales"

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `saleItems` | Mapping&lt;int32, Destiny.Components.Vendors.DestinyPublicVendorSaleItemComponent&gt; | — |

#### Destiny.Responses.DestinyCollectibleNodeDetailResponse

**Type:** object

Returns the detailed information about a Collectible Presentation Node and any Collectibles that are direct descendants.

| Property | Type | Description |
| --- | --- | --- |
| `collectibles` | SingleComponentResponseOfDestinyCollectiblesComponent | COMPONENT TYPE: Collectibles |
| `collectibleItemComponents` | DestinyItemComponentSetOfuint32 | Item components, keyed by the item hash of the items pointed at collectibles found under the requested Presentation Node. NOTE: I had a lot of hemming and hawing about whether these should be keyed by collectible hash or item hash... but ultimately having it be keyed by item hash meant that UI that already uses DestinyItemComponentSet data wouldn't have to have a special override to do the collectible -> item lookup once you delve into an item's details, and it also meant that you didn't have to remember that the Hash being used as the key for plugSets was different from the Hash being used for the other Dictionaries. As a result, using the Item Hash felt like the least crappy solution. We may all come to regret this decision. We will see. COMPONENT TYPE: [See inside the DestinyItemComponentSet contract for component types.] |

#### Destiny.Requests.Actions.DestinyActionRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `membershipType` | int32 | — |

#### Destiny.Requests.Actions.DestinyCharacterActionRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `characterId` | int64 | — |
| `membershipType` | int32 | — |

#### Destiny.Requests.Actions.DestinyItemActionRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `itemId` | int64 | The instance ID of the item for this action request. |
| `characterId` | int64 | — |
| `membershipType` | int32 | — |

#### Destiny.Requests.DestinyItemTransferRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `itemReferenceHash` | uint32 → DestinyInventoryItemDefinition | — |
| `stackSize` | int32 | — |
| `transferToVault` | boolean | — |
| `itemId` | int64 | The instance ID of the item for this action request. |
| `characterId` | int64 | — |
| `membershipType` | int32 | — |

#### Destiny.Requests.Actions.DestinyPostmasterTransferRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `itemReferenceHash` | uint32 → DestinyInventoryItemDefinition | — |
| `stackSize` | int32 | — |
| `itemId` | int64 | The instance ID of the item for this action request. |
| `characterId` | int64 | — |
| `membershipType` | int32 | — |

#### Destiny.DestinyEquipItemResults

**Type:** object

The results of a bulk Equipping operation performed through the Destiny API.

| Property | Type | Description |
| --- | --- | --- |
| `equipResults` | array&lt;Destiny.DestinyEquipItemResult&gt; | — |

#### Destiny.DestinyEquipItemResult

**Type:** object

The results of an Equipping operation performed through the Destiny API.

| Property | Type | Description |
| --- | --- | --- |
| `itemInstanceId` | int64 | The instance ID of the item in question (all items that can be equipped must, but definition, be Instanced and thus have an Instance ID that you can use to refer to them) |
| `equipStatus` | int32 | A PlatformErrorCodes enum indicating whether it succeeded, and if it failed why. |

#### Destiny.Requests.Actions.DestinyItemSetActionRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `itemIds` | array&lt;int64&gt; | — |
| `characterId` | int64 | — |
| `membershipType` | int32 | — |

#### Destiny.Requests.Actions.DestinyLoadoutActionRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `loadoutIndex` | int32 | The index of the loadout for this action request. |
| `characterId` | int64 | — |
| `membershipType` | int32 | — |

#### Destiny.Requests.Actions.DestinyLoadoutUpdateActionRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `colorHash` | uint32? | — |
| `iconHash` | uint32? | — |
| `nameHash` | uint32? | — |
| `loadoutIndex` | int32 | The index of the loadout for this action request. |
| `characterId` | int64 | — |
| `membershipType` | int32 | — |

#### Destiny.Requests.Actions.DestinyItemStateRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `state` | boolean | — |
| `itemId` | int64 | The instance ID of the item for this action request. |
| `characterId` | int64 | — |
| `membershipType` | int32 | — |

#### Destiny.Responses.InventoryChangedResponse

**Type:** object

A response containing all of the components for all requested vendors.

| Property | Type | Description |
| --- | --- | --- |
| `addedInventoryItems` | array&lt;Destiny.Entities.Items.DestinyItemComponent&gt; | Items that appeared in the inventory possibly as a result of an action. |
| `removedInventoryItems` | array&lt;Destiny.Entities.Items.DestinyItemComponent&gt; | Items that disappeared from the inventory possibly as a result of an action. |

#### Destiny.Responses.DestinyItemChangeResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `item` | Destiny.Responses.DestinyItemResponse | — |
| `addedInventoryItems` | array&lt;Destiny.Entities.Items.DestinyItemComponent&gt; | Items that appeared in the inventory possibly as a result of an action. |
| `removedInventoryItems` | array&lt;Destiny.Entities.Items.DestinyItemComponent&gt; | Items that disappeared from the inventory possibly as a result of an action. |

#### Destiny.Requests.Actions.DestinyInsertPlugsActionRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `actionToken` | string | Action token provided by the AwaGetActionToken API call. |
| `itemInstanceId` | int64 | The instance ID of the item having a plug inserted. Only instanced items can have sockets. |
| `plug` | Destiny.Requests.Actions.DestinyInsertPlugsRequestEntry | The plugs being inserted. |
| `characterId` | int64 | — |
| `membershipType` | int32 | — |

#### Destiny.Requests.Actions.DestinyInsertPlugsRequestEntry

**Type:** object

Represents all of the data related to a single plug to be inserted. Note that, while you *can* point to a socket that represents infusion, you will receive an error if you attempt to do so. Come on guys, let's play nice.

| Property | Type | Description |
| --- | --- | --- |
| `socketIndex` | int32 | The index into the socket array, which identifies the specific socket being operated on. We also need to know the socketArrayType in order to uniquely identify the socket. Don't point to or try to insert a plug into an infusion socket. It won't work. |
| `socketArrayType` | int32 | This property, combined with the socketIndex, tells us which socket we are referring to (since operations can be performed on both Intrinsic and "default" sockets, and they occupy different arrays in the Inventory Item Definition). I know, I know. Don't give me that look. |
| `plugItemHash` | uint32 | Plugs are never instanced (except in infusion). So with the hash alone, we should be able to: 1) Infer whether the player actually needs to have the item, or if it's a reusable plug 2) Perform any operation needed to use the Plug, including removing the plug item and running reward sheets. |

#### Destiny.Requests.Actions.DestinySocketArrayTypeEnumeration

**Enum** (`int32`)

If you look in the DestinyInventoryItemDefinition's "sockets" property, you'll see that there are two types of sockets: intrinsic, and "socketEntry." Unfortunately, because Intrinsic sockets are a whole separate array, it is no longer sufficient to know the index into that array to know which socket we're talking about. You have to know whether it's in the default "socketEntries" or if it's in the "intrinsic" list.

| Value | # | Description |
| --- | --- | --- |
| `Default` | 0 | — |
| `Intrinsic` | 1 | — |

#### Destiny.Requests.Actions.DestinyInsertPlugsFreeActionRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `plug` | Destiny.Requests.Actions.DestinyInsertPlugsRequestEntry | The plugs being inserted. |
| `itemId` | int64 | The instance ID of the item for this action request. |
| `characterId` | int64 | — |
| `membershipType` | int32 | — |

#### Destiny.HistoricalStats.DestinyPostGameCarnageReportData

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `period` | date-time | Date and time for the activity. |
| `startingPhaseIndex` | int32? | If this activity has "phases", this is the phase at which the activity was started. This value is only valid for activities before the Beyond Light expansion shipped. Subsequent activities will not have a valid value here. |
| `activityWasStartedFromBeginning` | boolean? | True if the activity was started from the beginning, if that information is available and the activity was played post Witch Queen release. |
| `activityDifficultyTier` | int32? | Difficulty tier index value for the activity. |
| `selectedSkullHashes` | array&lt;uint32&gt; | Collection of player-selected skull hashes active for the activity. |
| `activityDetails` | Destiny.HistoricalStats.DestinyHistoricalStatsActivity | Details about the activity. |
| `entries` | array&lt;Destiny.HistoricalStats.DestinyPostGameCarnageReportEntry&gt; | Collection of players and their data for this activity. |
| `teams` | array&lt;Destiny.HistoricalStats.DestinyPostGameCarnageReportTeamEntry&gt; | Collection of stats for the player in this activity. |

#### Destiny.HistoricalStats.DestinyHistoricalStatsActivity

**Type:** object

Summary information about the activity that was played.

| Property | Type | Description |
| --- | --- | --- |
| `referenceId` | uint32 → DestinyActivityDefinition | The unique hash identifier of the DestinyActivityDefinition that was played. If I had this to do over, it'd be named activityHash. Too late now. |
| `directorActivityHash` | uint32 → DestinyActivityDefinition | The unique hash identifier of the DestinyActivityDefinition that was played. |
| `instanceId` | int64 | The unique identifier for this *specific* match that was played. This value can be used to get additional data about this activity such as who else was playing via the GetPostGameCarnageReport endpoint. |
| `mode` | int32 | Indicates the most specific game mode of the activity that we could find. |
| `modes` | array&lt;int32&gt; | The list of all Activity Modes to which this activity applies, including aggregates. This will let you see, for example, whether the activity was both Clash and part of the Trials of the Nine event. |
| `isPrivate` | boolean | Whether or not the match was a private match. |
| `membershipType` | int32 | The Membership Type indicating the platform on which this match was played. |

#### Destiny.HistoricalStats.DestinyPostGameCarnageReportEntry

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `standing` | int32 | Standing of the player |
| `score` | Destiny.HistoricalStats.DestinyHistoricalStatsValue | Score of the player if available |
| `player` | Destiny.HistoricalStats.DestinyPlayer | Identity details of the player |
| `characterId` | int64 | ID of the player's character used in the activity. |
| `values` | Mapping&lt;string, Destiny.HistoricalStats.DestinyHistoricalStatsValue&gt; | Collection of stats for the player in this activity. |
| `extended` | Destiny.HistoricalStats.DestinyPostGameCarnageReportExtendedData | Extended data extracted from the activity blob. |

#### Destiny.HistoricalStats.DestinyHistoricalStatsValue

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `statId` | string | Unique ID for this stat |
| `basic` | Destiny.HistoricalStats.DestinyHistoricalStatsValuePair | Basic stat value. |
| `pga` | Destiny.HistoricalStats.DestinyHistoricalStatsValuePair | Per game average for the statistic, if applicable |
| `weighted` | Destiny.HistoricalStats.DestinyHistoricalStatsValuePair | Weighted value of the stat if a weight greater than 1 has been assigned. |
| `activityId` | int64? | When a stat represents the best, most, longest, fastest or some other personal best, the actual activity ID where that personal best was established is available on this property. |

#### Destiny.HistoricalStats.DestinyHistoricalStatsValuePair

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `value` | double | Raw value of the statistic |
| `displayValue` | string | Localized formated version of the value. |

#### Destiny.HistoricalStats.DestinyPlayer

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `destinyUserInfo` | User.UserInfoCard | Details about the player as they are known in game (platform display name, Destiny emblem) |
| `characterClass` | string | Class of the character if applicable and available. |
| `classHash` | uint32 → DestinyClassDefinition | — |
| `raceHash` | uint32 → DestinyRaceDefinition | — |
| `genderHash` | uint32 → DestinyGenderDefinition | — |
| `characterLevel` | int32 | Level of the character if available. Zero if it is not available. |
| `lightLevel` | int32 | Light Level of the character if available. Zero if it is not available. |
| `bungieNetUserInfo` | User.UserInfoCard | Details about the player as they are known on BungieNet. This will be undefined if the player has marked their credential private, or does not have a BungieNet account. |
| `clanName` | string | Current clan name for the player. This value may be null or an empty string if the user does not have a clan. |
| `clanTag` | string | Current clan tag for the player. This value may be null or an empty string if the user does not have a clan. |
| `emblemHash` | uint32 → DestinyInventoryItemDefinition | If we know the emblem's hash, this can be used to look up the player's emblem at the time of a match when receiving PGCR data, or otherwise their currently equipped emblem (if we are able to obtain it). |

#### Destiny.HistoricalStats.DestinyPostGameCarnageReportExtendedData

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `weapons` | array&lt;Destiny.HistoricalStats.DestinyHistoricalWeaponStats&gt; | List of weapons and their perspective values. |
| `values` | Mapping&lt;string, Destiny.HistoricalStats.DestinyHistoricalStatsValue&gt; | Collection of stats for the player in this activity. |
| `scoreboardValues` | Mapping&lt;string, Destiny.HistoricalStats.DestinyHistoricalStatsValue&gt; | Collection of stats from the player scoreboard in this activity. |

#### Destiny.HistoricalStats.DestinyHistoricalWeaponStats

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `referenceId` | uint32 → DestinyInventoryItemDefinition | The hash ID of the item definition that describes the weapon. |
| `values` | Mapping&lt;string, Destiny.HistoricalStats.DestinyHistoricalStatsValue&gt; | Collection of stats for the period. |

#### Destiny.HistoricalStats.DestinyPostGameCarnageReportTeamEntry

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `teamId` | int32 | Integer ID for the team. |
| `standing` | Destiny.HistoricalStats.DestinyHistoricalStatsValue | Team's standing relative to other teams. |
| `score` | Destiny.HistoricalStats.DestinyHistoricalStatsValue | Score earned by the team |
| `teamName` | string | Alpha or Bravo |

#### Destiny.Reporting.Requests.DestinyReportOffensePgcrRequest

**Type:** object

If you want to report a player causing trouble in a game, this request will let you report that player and the specific PGCR in which the trouble was caused, along with why. Please don't do this just because you dislike the person! I mean, I know people will do it anyways, but can you like take a good walk, or put a curse on them or something? Do me a solid and reconsider. Note that this request object doesn't have the actual PGCR ID nor your Account/Character ID in it. We will infer that information from your authentication information and the PGCR ID that you pass into the URL of the reporting endpoint itself.

| Property | Type | Description |
| --- | --- | --- |
| `reasonCategoryHashes` | array&lt;uint32&gt; → DestinyReportReasonCategoryDefinition | So you've decided to report someone instead of cursing them and their descendants. Well, okay then. This is the category or categorie(s) of infractions for which you are reporting the user. These are hash identifiers that map to DestinyReportReasonCategoryDefinition entries. |
| `reasonHashes` | array&lt;uint32&gt; | If applicable, provide a more specific reason(s) within the general category of problems provided by the reasonHash. This is also an identifier for a reason. All reasonHashes provided must be children of at least one the reasonCategoryHashes provided. |
| `offendingCharacterId` | int64 | Within the PGCR provided when calling the Reporting endpoint, this should be the character ID of the user that you thought was violating terms of use. They must exist in the PGCR provided. |

#### Destiny.Definitions.Reporting.DestinyReportReasonCategoryDefinition

**Object** · *(Manifest definition, table `ReportReasonCategories`)*

If you're going to report someone for a Terms of Service violation, you need to choose a category and reason for the report. This definition holds both the categories and the reasons within those categories, for simplicity and my own laziness' sake. Note tha this means that, to refer to a Reason by reasonHash, you need a combination of the reasonHash *and* the associated ReasonCategory's hash: there are some reasons defined under multiple categories.

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `reasons` | Mapping&lt;uint32, Destiny.Definitions.Reporting.DestinyReportReasonDefinition&gt; | The specific reasons for the report under this category. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Reporting.DestinyReportReasonDefinition

**Type:** object

A specific reason for being banned. Only accessible under the related category (DestinyReportReasonCategoryDefinition) under which it is shown. Note that this means that report reasons' reasonHash are not globally unique: and indeed, entries like "Other" are defined under most categories for example.

| Property | Type | Description |
| --- | --- | --- |
| `reasonHash` | uint32 | The identifier for the reason: they are only guaranteed unique under the Category in which they are found. |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |

#### Destiny.HistoricalStats.Definitions.DestinyHistoricalStatsDefinition

**Object** · *(Manifest definition, table `HistoricalStats`)*

| Property | Type | Description |
| --- | --- | --- |
| `statId` | string | Unique programmer friendly ID for this stat |
| `group` | int32 | Statistic group |
| `periodTypes` | array&lt;int32&gt; | Time periods the statistic covers |
| `modes` | array&lt;int32&gt; | Game modes where this statistic can be reported. |
| `category` | int32 | Category for the stat. |
| `statName` | string | Display name |
| `statNameAbbr` | string | Display name abbreviated |
| `statDescription` | string | Description of a stat if applicable. |
| `unitType` | int32 | Unit, if any, for the statistic |
| `iconImage` | string | Optional URI to an icon for the statistic |
| `mergeMethod` | int32? | Optional icon for the statistic |
| `unitLabel` | string | Localized Unit Name for the stat. |
| `weight` | int32 | Weight assigned to this stat indicating its relative impressiveness. |
| `medalTierHash` | uint32 → DestinyMedalTierDefinition? | The tier associated with this medal - be it implicitly or explicitly. |

#### Destiny.HistoricalStats.Definitions.DestinyStatsGroupTypeEnumeration

**Enum** (`int32`)

If the enum value is > 100, it is a "special" group that cannot be queried for directly (special cases apply to when they are returned, and are not relevant in general cases)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `General` | 1 | — |
| `Weapons` | 2 | — |
| `Medals` | 3 | — |
| `ReservedGroups` | 100 | This is purely to serve as the dividing line between filterable and un-filterable groups. Below this number is a group you can pass as a filter. Above it are groups used in very specific circumstances and not relevant for filtering. |
| `Leaderboard` | 101 | Only applicable while generating leaderboards. |
| `Activity` | 102 | These will *only* be consumed by GetAggregateStatsByActivity |
| `UniqueWeapon` | 103 | These are only consumed and returned by GetUniqueWeaponHistory |
| `Internal` | 104 | — |

#### Destiny.HistoricalStats.Definitions.PeriodType[]

**Type:** object

Type alias: `array<int32>`

#### Destiny.HistoricalStats.Definitions.DestinyActivityModeType[]

**Type:** object

Type alias: `array<int32>`

#### Destiny.HistoricalStats.Definitions.DestinyStatsCategoryTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Kills` | 1 | — |
| `Assists` | 2 | — |
| `Deaths` | 3 | — |
| `Criticals` | 4 | — |
| `KDa` | 5 | — |
| `KD` | 6 | — |
| `Score` | 7 | — |
| `Entered` | 8 | — |
| `TimePlayed` | 9 | — |
| `MedalWins` | 10 | — |
| `MedalGame` | 11 | — |
| `MedalSpecialKills` | 12 | — |
| `MedalSprees` | 13 | — |
| `MedalMultiKills` | 14 | — |
| `MedalAbilities` | 15 | — |

#### Destiny.HistoricalStats.Definitions.UnitTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Count` | 1 | Indicates the statistic is a simple count of something. |
| `PerGame` | 2 | Indicates the statistic is a per game average. |
| `Seconds` | 3 | Indicates the number of seconds |
| `Points` | 4 | Indicates the number of points earned |
| `Team` | 5 | Values represents a team ID |
| `Distance` | 6 | Values represents a distance (units to-be-determined) |
| `Percent` | 7 | Ratio represented as a whole value from 0 to 100. |
| `Ratio` | 8 | Ratio of something, shown with decimal places |
| `Boolean` | 9 | True or false |
| `WeaponType` | 10 | The stat is actually a weapon type. |
| `Standing` | 11 | Indicates victory, defeat, or something in between. |
| `Milliseconds` | 12 | Number of milliseconds some event spanned. For example, race time, or lap time. |
| `CompletionReason` | 13 | The value is a enumeration of the Completion Reason type. |

#### Destiny.HistoricalStats.Definitions.DestinyStatsMergeMethodEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Add` | 0 | When collapsing multiple instances of the stat together, add the values. |
| `Min` | 1 | When collapsing multiple instances of the stat together, take the lower value. |
| `Max` | 2 | When collapsing multiple instances of the stat together, take the higher value. |

#### Destiny.Definitions.DestinyMedalTierDefinition

**Object** · *(Manifest definition, table `MedalTiers`)*

An artificial construct of our own creation, to try and put some order on top of Medals and keep them from being one giant, unmanageable and unsorted blob of stats. Unfortunately, we haven't had time to do this evaluation yet in Destiny 2, so we're short on Medal Tiers. This will hopefully be updated over time, if Medals continue to exist.

| Property | Type | Description |
| --- | --- | --- |
| `tierName` | string | The name of the tier. |
| `order` | int32 | If you're rendering medals by tier, render them in this order (ascending) |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.HistoricalStats.DestinyLeaderboard

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `statId` | string | — |
| `entries` | array&lt;Destiny.HistoricalStats.DestinyLeaderboardEntry&gt; | — |

#### Destiny.HistoricalStats.DestinyLeaderboardEntry

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `rank` | int32 | Where this player ranks on the leaderboard. A value of 1 is the top rank. |
| `player` | Destiny.HistoricalStats.DestinyPlayer | Identity details of the player |
| `characterId` | int64 | ID of the player's best character for the reported stat. |
| `value` | Destiny.HistoricalStats.DestinyHistoricalStatsValue | Value of the stat for this player |

#### Destiny.HistoricalStats.DestinyLeaderboardResults

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `focusMembershipId` | int64? | Indicate the membership ID of the account that is the focal point of the provided leaderboards. |
| `focusCharacterId` | int64? | Indicate the character ID of the character that is the focal point of the provided leaderboards. May be null, in which case any character from the focus membership can appear in the provided leaderboards. |

#### Destiny.HistoricalStats.DestinyClanAggregateStat

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `mode` | int32 | The id of the mode of stats (allPvp, allPvE, etc) |
| `statId` | string | The id of the stat |
| `value` | Destiny.HistoricalStats.DestinyHistoricalStatsValue | Value of the stat for this player |

#### Destiny.Definitions.DestinyEntitySearchResult

**Type:** object

The results of a search for Destiny content. This will be improved on over time, I've been doing some experimenting to see what might be useful.

| Property | Type | Description |
| --- | --- | --- |
| `suggestedWords` | array&lt;string&gt; | A list of suggested words that might make for better search results, based on the text searched for. |
| `results` | SearchResultOfDestinyEntitySearchResultItem | The items found that are matches/near matches for the searched-for term, sorted by something vaguely resembling "relevance". Hopefully this will get better in the future. |

#### Destiny.Definitions.DestinyEntitySearchResultItem

**Type:** object

An individual Destiny Entity returned from the entity search.

| Property | Type | Description |
| --- | --- | --- |
| `hash` | uint32 | The hash identifier of the entity. You will use this to look up the DestinyDefinition relevant for the entity found. |
| `entityType` | string | The type of entity, returned as a string matching the DestinyDefinition's contract class name. You'll have to have your own mapping from class names to actually looking up those definitions in the manifest databases. |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | Basic display properties on the entity, so you don't have to look up the definition to show basic results for the item. |
| `weight` | double | The ranking value for sorting that we calculated using our relevance formula. This will hopefully get better with time and iteration. |

#### Destiny.HistoricalStats.Definitions.PeriodTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Daily` | 1 | — |
| `AllTime` | 2 | — |
| `Activity` | 3 | — |

#### Destiny.HistoricalStats.DestinyHistoricalStatsByPeriod

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `allTime` | Mapping&lt;string, Destiny.HistoricalStats.DestinyHistoricalStatsValue&gt; | — |
| `allTimeTier1` | Mapping&lt;string, Destiny.HistoricalStats.DestinyHistoricalStatsValue&gt; | — |
| `allTimeTier2` | Mapping&lt;string, Destiny.HistoricalStats.DestinyHistoricalStatsValue&gt; | — |
| `allTimeTier3` | Mapping&lt;string, Destiny.HistoricalStats.DestinyHistoricalStatsValue&gt; | — |
| `daily` | array&lt;Destiny.HistoricalStats.DestinyHistoricalStatsPeriodGroup&gt; | — |
| `monthly` | array&lt;Destiny.HistoricalStats.DestinyHistoricalStatsPeriodGroup&gt; | — |

#### Destiny.HistoricalStats.DestinyHistoricalStatsPeriodGroup

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `period` | date-time | Period for the group. If the stat periodType is day, then this will have a specific day. If the type is monthly, then this value will be the first day of the applicable month. This value is not set when the periodType is 'all time'. |
| `activityDetails` | Destiny.HistoricalStats.DestinyHistoricalStatsActivity | If the period group is for a specific activity, this property will be set. |
| `values` | Mapping&lt;string, Destiny.HistoricalStats.DestinyHistoricalStatsValue&gt; | Collection of stats for the period. |

#### Destiny.HistoricalStats.DestinyHistoricalStatsResults

**Type:** object

Type alias: `Mapping<string, Destiny.HistoricalStats.DestinyHistoricalStatsByPeriod>`

#### Destiny.HistoricalStats.DestinyHistoricalStatsAccountResult

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `mergedDeletedCharacters` | Destiny.HistoricalStats.DestinyHistoricalStatsWithMerged | — |
| `mergedAllCharacters` | Destiny.HistoricalStats.DestinyHistoricalStatsWithMerged | — |
| `characters` | array&lt;Destiny.HistoricalStats.DestinyHistoricalStatsPerCharacter&gt; | — |

#### Destiny.HistoricalStats.DestinyHistoricalStatsWithMerged

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | Mapping&lt;string, Destiny.HistoricalStats.DestinyHistoricalStatsByPeriod&gt; | — |
| `merged` | Destiny.HistoricalStats.DestinyHistoricalStatsByPeriod | — |

#### Destiny.HistoricalStats.DestinyHistoricalStatsPerCharacter

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `characterId` | int64 | — |
| `deleted` | boolean | — |
| `results` | Mapping&lt;string, Destiny.HistoricalStats.DestinyHistoricalStatsByPeriod&gt; | — |
| `merged` | Destiny.HistoricalStats.DestinyHistoricalStatsByPeriod | — |

#### Destiny.HistoricalStats.DestinyActivityHistoryResults

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `activities` | array&lt;Destiny.HistoricalStats.DestinyHistoricalStatsPeriodGroup&gt; | List of activities, the most recent activity first. |

#### Destiny.HistoricalStats.DestinyHistoricalWeaponStatsData

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `weapons` | array&lt;Destiny.HistoricalStats.DestinyHistoricalWeaponStats&gt; | List of weapons and their perspective values. |

#### Destiny.HistoricalStats.DestinyAggregateActivityResults

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `activities` | array&lt;Destiny.HistoricalStats.DestinyAggregateActivityStats&gt; | List of all activities the player has participated in. |

#### Destiny.HistoricalStats.DestinyAggregateActivityStats

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `activityHash` | uint32 → DestinyActivityDefinition | Hash ID that can be looked up in the DestinyActivityTable. |
| `values` | Mapping&lt;string, Destiny.HistoricalStats.DestinyHistoricalStatsValue&gt; | Collection of stats for the player in this activity. |

#### Destiny.Milestones.DestinyMilestoneContent

**Type:** object

Represents localized, extended content related to Milestones. This is intentionally returned by a separate endpoint and not with Character-level Milestone data because we do not put localized data into standard Destiny responses, both for brevity of response and for caching purposes. If you really need this data, hit the Milestone Content endpoint.

| Property | Type | Description |
| --- | --- | --- |
| `about` | string | The "About this Milestone" text from the Firehose. |
| `status` | string | The Current Status of the Milestone, as driven by the Firehose. |
| `tips` | array&lt;string&gt; | A list of tips, provided by the Firehose. |
| `itemCategories` | array&lt;Destiny.Milestones.DestinyMilestoneContentItemCategory&gt; | If DPS has defined items related to this Milestone, they can categorize those items in the Firehose. That data will then be returned as item categories here. |

#### Destiny.Milestones.DestinyMilestoneContentItemCategory

**Type:** object

Part of our dynamic, localized Milestone content is arbitrary categories of items. These are built in our content management system, and thus aren't the same as programmatically generated rewards.

| Property | Type | Description |
| --- | --- | --- |
| `title` | string | — |
| `itemHashes` | array&lt;uint32&gt; → DestinyInventoryItemDefinition | — |

#### Destiny.Milestones.DestinyPublicMilestone

**Type:** object

Information about milestones, presented in a character state-agnostic manner. Combine this data with DestinyMilestoneDefinition to get a full picture of the milestone, which is basically a checklist of things to do in the game. Think of this as GetPublicAdvisors 3.0, for those who used the Destiny 1 API.

| Property | Type | Description |
| --- | --- | --- |
| `milestoneHash` | uint32 → DestinyMilestoneDefinition | The hash identifier for the milestone. Use it to look up the DestinyMilestoneDefinition for static data about the Milestone. |
| `availableQuests` | array&lt;Destiny.Milestones.DestinyPublicMilestoneQuest&gt; | A milestone not need have even a single quest, but if there are active quests they will be returned here. |
| `activities` | array&lt;Destiny.Milestones.DestinyPublicMilestoneChallengeActivity&gt; | — |
| `vendorHashes` | array&lt;uint32&gt; | Sometimes milestones - or activities active in milestones - will have relevant vendors. These are the vendors that are currently relevant. Deprecated, already, for the sake of the new "vendors" property that has more data. What was I thinking. |
| `vendors` | array&lt;Destiny.Milestones.DestinyPublicMilestoneVendor&gt; | This is why we can't have nice things. This is the ordered list of vendors to be shown that relate to this milestone, potentially along with other interesting data. |
| `startDate` | date-time? | If known, this is the date when the Milestone started/became active. |
| `endDate` | date-time? | If known, this is the date when the Milestone will expire/recycle/end. |
| `order` | int32 | Used for ordering milestones in a display to match how we order them in BNet. May pull from static data, or possibly in the future from dynamic information. |

#### Destiny.Milestones.DestinyPublicMilestoneQuest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `questItemHash` | uint32 → DestinyMilestoneDefinition | Quests are defined as Items in content. As such, this is the hash identifier of the DestinyInventoryItemDefinition that represents this quest. It will have pointers to all of the steps in the quest, and display information for the quest (title, description, icon etc) Individual steps will be referred to in the Quest item's DestinyInventoryItemDefinition.setData property, and themselves are Items with their own renderable data. |
| `activity` | Destiny.Milestones.DestinyPublicMilestoneActivity | A milestone need not have an active activity, but if there is one it will be returned here, along with any variant and additional information. |
| `challenges` | array&lt;Destiny.Milestones.DestinyPublicMilestoneChallenge&gt; | For the given quest there could be 0-to-Many challenges: mini quests that you can perform in the course of doing this quest, that may grant you rewards and benefits. |

#### Destiny.Milestones.DestinyPublicMilestoneActivity

**Type:** object

A milestone may have one or more conceptual Activities associated with it, and each of those conceptual activities could have a variety of variants, modes, tiers, what-have-you. Our attempts to determine what qualifies as a conceptual activity are, unfortunately, janky. So if you see missing modes or modes that don't seem appropriate to you, let us know and I'll buy you a beer if we ever meet up in person.

| Property | Type | Description |
| --- | --- | --- |
| `activityHash` | uint32 → DestinyActivityDefinition | The hash identifier of the activity that's been chosen to be considered the canonical "conceptual" activity definition. This may have many variants, defined herein. |
| `modifierHashes` | array&lt;uint32&gt; → DestinyActivityModifierDefinition | The activity may have 0-to-many modifiers: if it does, this will contain the hashes to the DestinyActivityModifierDefinition that defines the modifier being applied. |
| `variants` | array&lt;Destiny.Milestones.DestinyPublicMilestoneActivityVariant&gt; | Every relevant variation of this conceptual activity, including the conceptual activity itself, have variants defined here. |
| `activityModeHash` | uint32 → DestinyActivityModeDefinition? | The hash identifier of the most specific Activity Mode under which this activity is played. This is useful for situations where the activity in question is - for instance - a PVP map, but it's not clear what mode the PVP map is being played under. If it's a playlist, this will be less specific: but hopefully useful in some way. |
| `activityModeType` | int32? | The enumeration equivalent of the most specific Activity Mode under which this activity is played. |

#### Destiny.Milestones.DestinyPublicMilestoneActivityVariant

**Type:** object

Represents a variant of an activity that's relevant to a milestone.

| Property | Type | Description |
| --- | --- | --- |
| `activityHash` | uint32 → DestinyActivityDefinition | The hash identifier of this activity variant. Examine the activity's definition in the Manifest database to determine what makes it a distinct variant. Usually it will be difficulty level or whether or not it is a guided game variant of the activity, but theoretically it could be distinguished in any arbitrary way. |
| `activityModeHash` | uint32 → DestinyActivityModeDefinition? | The hash identifier of the most specific Activity Mode under which this activity is played. This is useful for situations where the activity in question is - for instance - a PVP map, but it's not clear what mode the PVP map is being played under. If it's a playlist, this will be less specific: but hopefully useful in some way. |
| `activityModeType` | int32? | The enumeration equivalent of the most specific Activity Mode under which this activity is played. |

#### Destiny.Milestones.DestinyPublicMilestoneChallenge

**Type:** object

A Milestone can have many Challenges. Challenges are just extra Objectives that provide a fun way to mix-up play and provide extra rewards.

| Property | Type | Description |
| --- | --- | --- |
| `objectiveHash` | uint32 → DestinyObjectiveDefinition | The objective for the Challenge, which should have human-readable data about what needs to be done to accomplish the objective. Use this hash to look up the DestinyObjectiveDefinition. |
| `activityHash` | uint32 → DestinyActivityDefinition? | IF the Objective is related to a specific Activity, this will be that activity's hash. Use it to look up the DestinyActivityDefinition for additional data to show. |

#### Destiny.Milestones.DestinyPublicMilestoneChallengeActivity

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `activityHash` | uint32 → DestinyActivityDefinition | — |
| `challengeObjectiveHashes` | array&lt;uint32&gt; | — |
| `modifierHashes` | array&lt;uint32&gt; → DestinyActivityModifierDefinition | If the activity has modifiers, this will be the list of modifiers that all variants have in common. Perform lookups against DestinyActivityModifierDefinition which defines the modifier being applied to get at the modifier data. Note that, in the DestinyActivityDefinition, you will see many more modifiers than this being referred to: those are all *possible* modifiers for the activity, not the active ones. Use only the active ones to match what's really live. |
| `loadoutRequirementIndex` | int32? | If returned, this is the index into the DestinyActivityDefinition's "loadouts" property, indicating the currently active loadout requirements. |
| `phaseHashes` | array&lt;uint32&gt; | The ordered list of phases for this activity, if any. Note that we have no human readable info for phases, nor any entities to relate them to: relating these hashes to something human readable is up to you unfortunately. |
| `booleanActivityOptions` | Mapping&lt;uint32, boolean&gt; | The set of activity options for this activity, keyed by an identifier that's unique for this activity (not guaranteed to be unique between or across all activities, though should be unique for every *variant* of a given *conceptual* activity: for instance, the original D2 Raid has many variant DestinyActivityDefinitions. While other activities could potentially have the same option hashes, for any given D2 base Raid variant the hash will be unique). As a concrete example of this data, the hashes you get for Raids will correspond to the currently active "Challenge Mode". We have no human readable information for this data, so it's up to you if you want to associate it with such info to show it. |

#### Destiny.Milestones.DestinyPublicMilestoneVendor

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `vendorHash` | uint32 → DestinyVendorDefinition | The hash identifier of the Vendor related to this Milestone. You can show useful things from this, such as thier Faction icon or whatever you might care about. |
| `previewItemHash` | uint32 → DestinyInventoryItemDefinition? | If this vendor is featuring a specific item for this event, this will be the hash identifier of that item. I'm taking bets now on how long we go before this needs to be a list or some other, more complex representation instead and I deprecate this too. I'm going to go with 5 months. Calling it now, 2017-09-14 at 9:46pm PST. |

#### Destiny.Advanced.AwaInitializeResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `correlationId` | string | ID used to get the token. Present this ID to the user as it will identify this specific request on their device. |
| `sentToSelf` | boolean | True if the PUSH message will only be sent to the device that made this request. |

#### Destiny.Advanced.AwaPermissionRequested

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `type` | int32 | Type of advanced write action. |
| `affectedItemId` | int64? | Item instance ID the action shall be applied to. This is optional for all but a new AwaType values. Rule of thumb is to provide the item instance ID if one is available. |
| `membershipType` | int32 | Destiny membership type of the account to modify. |
| `characterId` | int64? | Destiny character ID, if applicable, that will be affected by the action. |

#### Destiny.Advanced.AwaTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `InsertPlugs` | 1 | Insert plugs into sockets. |

#### Destiny.Advanced.AwaUserResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `selection` | int32 | Indication of the selection the user has made (Approving or rejecting the action) |
| `correlationId` | string | Correlation ID of the request |
| `nonce` | array&lt;byte&gt; | Secret nonce received via the PUSH notification. |

#### Destiny.Advanced.AwaUserSelectionEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Rejected` | 1 | — |
| `Approved` | 2 | — |

#### Destiny.Advanced.AwaAuthorizationResult

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `userSelection` | int32 | Indication of how the user responded to the request. If the value is "Approved" the actionToken will contain the token that can be presented when performing the advanced write action. |
| `responseReason` | int32 | — |
| `developerNote` | string | Message to the app developer to help understand the response. |
| `actionToken` | string | Credential used to prove the user authorized an advanced write action. |
| `maximumNumberOfUses` | int32 | This token may be used to perform the requested action this number of times, at a maximum. If this value is 0, then there is no limit. |
| `validUntil` | date-time? | Time, UTC, when token expires. |
| `type` | int32 | Advanced Write Action Type from the permission request. |
| `membershipType` | int32 | MembershipType from the permission request. |

#### Destiny.Advanced.AwaResponseReasonEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Answered` | 1 | User provided an answer |
| `TimedOut` | 2 | The HTTP request timed out, a new request may be made and an answer may still be provided. |
| `Replaced` | 3 | This request was replaced by another request. |

#### Destiny.Activities.DestinyPublicActivityStatus

**Type:** object

Represents the public-facing status of an activity: any data about what is currently active in the Activity, regardless of an individual character's progress in it.

| Property | Type | Description |
| --- | --- | --- |
| `challengeObjectiveHashes` | array&lt;uint32&gt; → DestinyObjectiveDefinition | Active Challenges for the activity, if any - represented as hashes for DestinyObjectiveDefinitions. |
| `modifierHashes` | array&lt;uint32&gt; → DestinyActivityModifierDefinition | The active modifiers on this activity, if any - represented as hashes for DestinyActivityModifierDefinitions. |
| `rewardTooltipItems` | array&lt;Destiny.DestinyItemQuantity&gt; | If the activity itself provides any specific "mock" rewards, this will be the items and their quantity. Why "mock", you ask? Because these are the rewards as they are represented in the tooltip of the Activity. These are often pointers to fake items that look good in a tooltip, but represent an abstract concept of what you will get for a reward rather than the specific items you may obtain. |

#### Destiny.Definitions.Common.DestinyGlobalConstantsDefinition

**Object** · *(Manifest definition, table `GlobalConstants`)*

| Property | Type | Description |
| --- | --- | --- |
| `pathfinderConstants` | Destiny.Definitions.Common.DestinyPathfinderConstantsDefinition | Assorted constants for Pathfinder objectives |
| `collectionsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `collectionBadgesRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `activeTriumphsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `activeSealsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `legacyTriumphsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `legacySealsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `medalsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `exoticCatalystsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `loreRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `metricsRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `craftingRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `guardianRanksRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `seasonalHubEventCardHash` | uint32 → DestinyEventCardDefinition | — |
| `destinyRewardPassRankSealImages` | Destiny.Definitions.Common.DestinyRewardPassRankSealImages | — |
| `destinySeasonalHubRankIconImages` | Destiny.Definitions.Common.DestinySeasonalHubRankIconImages | — |
| `armorArchetypePlugSetHash` | uint32 → DestinyPlugSetDefinition | — |
| `featuredItemsListHash` | uint32 → DestinyItemFilterDefinition | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Common.DestinyPathfinderConstantsDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `thePaleHeartPathfinderRootNodeHash` | uint32 → DestinyPresentationNodeDefinition | Pathfinder root node for The Pale Heart |
| `allPathfinderRootNodeHashes` | array&lt;uint32&gt; → DestinyPresentationNodeDefinition | Root presentation nodes for all currently valid Pathfinder boards |
| `pathfinderTreeTiers` | Mapping&lt;uint32, uint32&gt; | The current shape of Pathfinder boards, where a Pathfinder board is stored as as flat list of Records. The key of this dictionary is the index at which a tier starts, and the value is the total number of objectives in the tier. |
| `pathfinderTopology` | Mapping&lt;uint32, array&gt; | The topology of the Pathfinder board. The key is the index of the Record in the Pathfinder board, and the value is a list of the indices of Records that are connected to the Key Record. Using this topology, clients can ascertain if a Record can be unlocked, by checking if the objective of any connected Record has been completed and/or claimed. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Common.DestinyRewardPassRankSealImages

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `rewardPassRankSealImagePath` | string | — |
| `rewardPassRankSealPremiumImagePath` | string | — |
| `rewardPassRankSealPrestigeImagePath` | string | — |
| `rewardPassRankSealPremiumPrestigeImagePath` | string | — |

#### Destiny.Definitions.Common.DestinySeasonalHubRankIconImages

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `seasonalHubRankIconUnearned` | string | — |
| `seasonalHubRankIconEarning` | string | — |
| `seasonalHubRankIconActive` | string | — |

#### Destiny.Definitions.Inventory.DestinyItemFilterDefinition

**Object** · *(Manifest definition, table `itemFilters`)*

Lists of items that can be used for a variety of purposes, including featuring them as new gear

| Property | Type | Description |
| --- | --- | --- |
| `allowedItems` | array&lt;uint32&gt; → DestinyInventoryItemDefinition | The items in this set |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Loadouts.DestinyLoadoutConstantsDefinition

**Object** · *(Manifest definition, table `LoadoutConstants`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `whiteIconImagePath` | string | This is the same icon as the one in the display properties, offered here as well with a more descriptive name. |
| `blackIconImagePath` | string | This is a color-inverted version of the whiteIconImagePath. |
| `loadoutCountPerCharacter` | int32 | The maximum number of loadouts available to each character. The loadouts component API response can return fewer loadouts than this, as more loadouts are unlocked by reaching higher Guardian Ranks. |
| `loadoutPreviewFilterOutSocketCategoryHashes` | array&lt;uint32&gt; → DestinySocketCategoryDefinition | A list of the socket category hashes to be filtered out of loadout item preview displays. |
| `loadoutPreviewFilterOutSocketTypeHashes` | array&lt;uint32&gt; → DestinySocketTypeDefinition | A list of the socket type hashes to be filtered out of loadout item preview displays. |
| `loadoutNameHashes` | array&lt;uint32&gt; → DestinyLoadoutNameDefinition | A list of the loadout name hashes in index order, for convenience. |
| `loadoutIconHashes` | array&lt;uint32&gt; → DestinyLoadoutIconDefinition | A list of the loadout icon hashes in index order, for convenience. |
| `loadoutColorHashes` | array&lt;uint32&gt; → DestinyLoadoutColorDefinition | A list of the loadout color hashes in index order, for convenience. |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.GuardianRanks.DestinyGuardianRankConstantsDefinition

**Object** · *(Manifest definition, table `GuardianRankConstants`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `rankCount` | int32 | — |
| `guardianRankHashes` | array&lt;uint32&gt; → DestinyGuardianRankDefinition | — |
| `rootNodeHash` | uint32 → DestinyPresentationNodeDefinition | — |
| `iconBackgrounds` | Destiny.Definitions.GuardianRanks.DestinyGuardianRankIconBackgroundsDefinition | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.GuardianRanks.DestinyGuardianRankIconBackgroundsDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `backgroundEmptyBorderedImagePath` | string | — |
| `backgroundEmptyBlueGradientBorderedImagePath` | string | — |
| `backgroundFilledBlueBorderedImagePath` | string | — |
| `backgroundFilledBlueGradientBorderedImagePath` | string | — |
| `backgroundFilledBlueLowAlphaImagePath` | string | — |
| `backgroundFilledBlueMediumAlphaImagePath` | string | — |
| `backgroundFilledGrayMediumAlphaBorderedImagePath` | string | — |
| `backgroundFilledGrayHeavyAlphaBorderedImagePath` | string | — |
| `backgroundFilledWhiteMediumAlphaImagePath` | string | — |
| `backgroundFilledWhiteImagePath` | string | — |
| `backgroundPlateWhiteImagePath` | string | — |
| `backgroundPlateBlackImagePath` | string | — |
| `backgroundPlateBlackAlphaImagePath` | string | — |

#### Destiny.Definitions.FireteamFinder.DestinyFireteamFinderConstantsDefinition

**Object** · *(Manifest definition, table `FireteamFinderConstants`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `fireteamFinderActivityGraphRootCategoryHashes` | array&lt;uint32&gt; → DestinyFireteamFinderActivityGraphDefinition | — |
| `allFireteamFinderActivityHashes` | array&lt;uint32&gt; → DestinyActivityDefinition | — |
| `guardianOathDisplayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `guardianOathTenets` | array&lt;Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition&gt; | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.FireteamFinder.DestinyFireteamFinderActivityGraphDefinition

**Object** · *(Manifest definition, table `FireteamFinderActivityGraphs`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `color` | Destiny.Misc.DestinyColor | — |
| `isPlayerElectedDifficultyNode` | boolean | — |
| `parentHash` | uint32 → DestinyFireteamFinderActivityGraphDefinition? | — |
| `children` | array&lt;uint32&gt; → DestinyFireteamFinderActivityGraphDefinition | — |
| `selfAndAllDescendantHashes` | array&lt;uint32&gt; → DestinyFireteamFinderActivityGraphDefinition | — |
| `relatedActivitySetHashes` | array&lt;uint32&gt; → DestinyFireteamFinderActivitySetDefinition | — |
| `specificActivitySetHash` | uint32 → DestinyFireteamFinderActivitySetDefinition? | — |
| `relatedActivityHashes` | array&lt;uint32&gt; → DestinyActivityDefinition | — |
| `relatedDirectorNodes` | array&lt;Destiny.Definitions.FireteamFinder.DestinyActivityGraphReference&gt; | — |
| `relatedInteractableActivities` | array&lt;Destiny.Definitions.FireteamFinder.DestinyActivityInteractableReference&gt; | — |
| `relatedLocationHashes` | array&lt;uint32&gt; → DestinyLocationDefinition | — |
| `sortMatchmadeActivitiesToFront` | boolean | — |
| `enabledOnTreeTypesListEnum` | array&lt;int32&gt; | — |
| `activityTreeChildSortMode` | int32 | — |
| `sortPriority` | int32? | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.FireteamFinder.DestinyActivityGraphReference

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `activityGraphHash` | uint32 → DestinyActivityGraphDefinition | — |

#### Destiny.DestinyActivityTreeTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `FireteamFinder` | 0 | — |
| `Curator` | 1 | — |
| `EventHome` | 2 | — |
| `SeasonHome` | 3 | — |
| `Count` | 4 | — |

#### Destiny.DestinyActivityTreeChildSortModeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Investment` | 0 | — |
| `FocusFirst` | 1 | — |
| `BonusAndFocusFirst` | 2 | — |

#### Destiny.Definitions.FireteamFinder.DestinyFireteamFinderActivitySetDefinition

**Object** · *(Manifest definition, table `FireteamFinderActivitySets`)*

| Property | Type | Description |
| --- | --- | --- |
| `maximumPartySize` | int32 | — |
| `optionHashes` | array&lt;uint32&gt; → DestinyFireteamFinderOptionDefinition | — |
| `labelHashes` | array&lt;uint32&gt; → DestinyFireteamFinderLabelDefinition | — |
| `activityGraphHashes` | array&lt;uint32&gt; → DestinyFireteamFinderActivityGraphDefinition | — |
| `activityHashes` | array&lt;uint32&gt; → DestinyActivityDefinition | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionDefinition

**Object** · *(Manifest definition, table `FireteamFinderOptions`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `descendingSortPriority` | int32 | — |
| `groupHash` | uint32 → DestinyFireteamFinderOptionGroupDefinition | — |
| `codeOptionType` | int32 | — |
| `availability` | int32 | — |
| `visibility` | int32 | — |
| `uiDisplayStyle` | string | — |
| `creatorSettings` | Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionCreatorSettings | — |
| `searcherSettings` | Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionSearcherSettings | — |
| `values` | Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionValues | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.FireteamFinderCodeOptionTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `ApplicationOnly` | 1 | — |
| `OnlineOnly` | 2 | — |
| `PlayerCount` | 3 | — |
| `Title` | 4 | — |
| `Tags` | 5 | — |
| `FinderActivityGraph` | 6 | — |
| `MicrophoneRequired` | 7 | — |

#### Destiny.FireteamFinderOptionAvailabilityEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `CreateListingBuilder` | 1 | — |
| `SearchListingBuilder` | 2 | — |
| `ListingViewer` | 4 | — |
| `LobbyViewer` | 8 | — |

#### Destiny.FireteamFinderOptionVisibilityEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Always` | 0 | — |
| `ShowWhenChangedFromDefault` | 1 | — |

#### Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionCreatorSettings

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `control` | Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionSettingsControl | — |

#### Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionSettingsControl

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `type` | int32 | — |
| `minSelectedItems` | int32 | — |
| `maxSelectedItems` | int32 | — |

#### Destiny.FireteamFinderOptionControlTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `ValueCollection` | 1 | — |
| `RadioButton` | 2 | — |

#### Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionSearcherSettings

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `control` | Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionSettingsControl | — |
| `searchFilterType` | int32 | — |

#### Destiny.FireteamFinderOptionSearchFilterTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `All` | 1 | — |
| `Any` | 2 | — |
| `InRangeInclusive` | 3 | — |
| `InRangeExclusive` | 4 | — |
| `GreaterThan` | 5 | — |
| `GreaterThanOrEqualTo` | 6 | — |
| `LessThan` | 7 | — |
| `LessThanOrEqualTo` | 8 | — |

#### Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionValues

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `optionalNull` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `optionalFormatString` | string | — |
| `displayFormatType` | int32 | — |
| `type` | int32 | — |
| `valueDefinitions` | array&lt;Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionValueDefinition&gt; | — |

#### Destiny.FireteamFinderOptionDisplayFormatEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Text` | 0 | — |
| `Integer` | 1 | — |
| `Bool` | 2 | — |
| `FormatString` | 3 | — |

#### Destiny.FireteamFinderOptionValueProviderTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Values` | 1 | — |
| `PlayerCount` | 2 | — |
| `FireteamFinderLabels` | 3 | — |
| `FireteamFinderActivityGraph` | 4 | — |
| `FireteamFinderUIActivityTree` | 5 | — |

#### Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionValueDefinition

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `value` | uint32 | — |
| `flags` | int32 | — |

#### Destiny.FireteamFinderOptionValueFlagsEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `CreateListingDefaultValue` | 1 | — |
| `SearchFilterDefaultValue` | 2 | — |

#### Destiny.Definitions.FireteamFinder.DestinyFireteamFinderOptionGroupDefinition

**Object** · *(Manifest definition, table `FireteamFinderOptionGroups`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `descendingSortPriority` | int32 | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.FireteamFinder.DestinyFireteamFinderLabelDefinition

**Object** · *(Manifest definition, table `FireteamFinderLabels`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `descendingSortPriority` | int32 | — |
| `groupHash` | uint32 → DestinyFireteamFinderLabelGroupDefinition | — |
| `allowInFields` | int32 | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.FireteamFinderLabelFieldTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Title` | 0 | — |
| `Label` | 1 | — |

#### Destiny.Definitions.FireteamFinder.DestinyFireteamFinderLabelGroupDefinition

**Object** · *(Manifest definition, table `FireteamFinderLabelGroups`)*

| Property | Type | Description |
| --- | --- | --- |
| `displayProperties` | Destiny.Definitions.Common.DestinyDisplayPropertiesDefinition | — |
| `descendingSortPriority` | int32 | — |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

#### Destiny.Definitions.Items.DestinyInventoryItemConstantsDefinition

**Object** · *(Manifest definition, table `InventoryItemConstants`)*

| Property | Type | Description |
| --- | --- | --- |
| `gearTierOverlayImagePaths` | array&lt;string&gt; | Gear tier overlay images |
| `watermarkDropShadowPath` | string | Watermark drop shadow |
| `craftedBackgroundPath` | string | Reverse drop shadow for crafted icon identifier |
| `featuredItemFlagPath` | string | Teal flag for featured item watermarks |
| `masterworkOverlayPath` | string | Gold masterwork glow for non-Exotic items |
| `masterworkExoticOverlayPath` | string | Gold masterwork glow for Exotic items |
| `masterworkBorderedOverlayPath` | string | Gold masterwork glow for non-Exotic Items, with a gold border |
| `masterworkExoticBorderedOverlayPath` | string | Gold masterwork glow for Exotic items, with a gold border |
| `craftedOverlayPath` | string | Crafted weapon overlay path |
| `enhancedItemOverlayPath` | string | Enhanced item overlay |
| `holofoilBackgroundOverlayPath` | string | Layer between item and color background to denote holofoil status, introduced in v736 |
| `holofoil900BackgroundOverlayPath` | string | Layer between item and color background to denote holofoil status, introduced in v900 |
| `holofoil900AnimatedBackgroundOverlayPath` | string | Layer between item and color background to denote holofoil status, introduced in v900, animated |
| `universalOrnamentBackgroundOverlayPath` | string | Layer between item and color background to denote universal ornament status |
| `universalOrnamentLegendaryBackgroundOverlayPath` | string | Layer between a legendary item and its color background to denote universal ornament status |
| `universalOrnamentExoticBackgroundOverlayPath` | string | Layer between an exotic item and its color background to denote universal ornament status |
| `hash` | uint32 | The unique identifier for this entity. Guaranteed to be unique for the type of entity, but not globally. When entities refer to each other in Destiny content, it is this hash that they are referring to. |
| `index` | int32 | The index of the entity as it was found in the investment tables. |
| `redacted` | boolean | If this is true, then there is an entity with this identifier/type combination, but BNet is not yet allowed to show it. Sorry! |

### Namespace: Entities

#### Entities.EntityActionResult

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `entityId` | int64 | — |
| `result` | int32 | — |

### Namespace: Exceptions

#### Exceptions.PlatformErrorCodesEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Success` | 1 | — |
| `TransportException` | 2 | — |
| `UnhandledException` | 3 | — |
| `NotImplemented` | 4 | — |
| `SystemDisabled` | 5 | — |
| `FailedToLoadAvailableLocalesConfiguration` | 6 | — |
| `ParameterParseFailure` | 7 | — |
| `ParameterInvalidRange` | 8 | — |
| `BadRequest` | 9 | — |
| `AuthenticationInvalid` | 10 | — |
| `DataNotFound` | 11 | — |
| `InsufficientPrivileges` | 12 | — |
| `Duplicate` | 13 | — |
| `UnknownSqlResult` | 14 | — |
| `ValidationError` | 15 | — |
| `ValidationMissingFieldError` | 16 | — |
| `ValidationInvalidInputError` | 17 | — |
| `InvalidParameters` | 18 | — |
| `ParameterNotFound` | 19 | — |
| `UnhandledHttpException` | 20 | — |
| `NotFound` | 21 | — |
| `WebAuthModuleAsyncFailed` | 22 | — |
| `InvalidReturnValue` | 23 | — |
| `UserBanned` | 24 | — |
| `InvalidPostBody` | 25 | — |
| `MissingPostBody` | 26 | — |
| `ExternalServiceTimeout` | 27 | — |
| `ValidationLengthError` | 28 | — |
| `ValidationRangeError` | 29 | — |
| `JsonDeserializationError` | 30 | — |
| `ThrottleLimitExceeded` | 31 | — |
| `ValidationTagError` | 32 | — |
| `ValidationProfanityError` | 33 | — |
| `ValidationUrlFormatError` | 34 | — |
| `ThrottleLimitExceededMinutes` | 35 | — |
| `ThrottleLimitExceededMomentarily` | 36 | — |
| `ThrottleLimitExceededSeconds` | 37 | — |
| `ExternalServiceUnknown` | 38 | — |
| `ValidationWordLengthError` | 39 | — |
| `ValidationInvisibleUnicode` | 40 | — |
| `ValidationBadNames` | 41 | — |
| `ExternalServiceFailed` | 42 | — |
| `ServiceRetired` | 43 | — |
| `UnknownSqlException` | 44 | — |
| `UnsupportedLocale` | 45 | — |
| `InvalidPageNumber` | 46 | — |
| `MaximumPageSizeExceeded` | 47 | — |
| `ServiceUnsupported` | 48 | — |
| `ValidationMaximumUnicodeCombiningCharacters` | 49 | — |
| `ValidationMaximumSequentialCarriageReturns` | 50 | — |
| `PerEndpointRequestThrottleExceeded` | 51 | — |
| `AuthContextCacheAssertion` | 52 | — |
| `ExPlatformStringValidationError` | 53 | — |
| `PerApplicationThrottleExceeded` | 54 | — |
| `PerApplicationAnonymousThrottleExceeded` | 55 | — |
| `PerApplicationAuthenticatedThrottleExceeded` | 56 | — |
| `PerUserThrottleExceeded` | 57 | — |
| `PayloadSignatureVerificationFailure` | 58 | — |
| `InvalidServiceAuthContext` | 59 | — |
| `FailedMinimumAgeCheck` | 60 | — |
| `ObsoleteCredentialType` | 89 | — |
| `UnableToUnPairMobileApp` | 90 | — |
| `UnableToPairMobileApp` | 91 | — |
| `CannotUseMobileAuthWithNonMobileProvider` | 92 | — |
| `MissingDeviceCookie` | 93 | — |
| `FacebookTokenExpired` | 94 | — |
| `AuthTicketRequired` | 95 | — |
| `CookieContextRequired` | 96 | — |
| `UnknownAuthenticationError` | 97 | — |
| `BungieNetAccountCreationRequired` | 98 | — |
| `WebAuthRequired` | 99 | — |
| `ContentUnknownSqlResult` | 100 | — |
| `ContentNeedUniquePath` | 101 | — |
| `ContentSqlException` | 102 | — |
| `ContentNotFound` | 103 | — |
| `ContentSuccessWithTagAddFail` | 104 | — |
| `ContentSearchMissingParameters` | 105 | — |
| `ContentInvalidId` | 106 | — |
| `ContentPhysicalFileDeletionError` | 107 | — |
| `ContentPhysicalFileCreationError` | 108 | — |
| `ContentPerforceSubmissionError` | 109 | — |
| `ContentPerforceInitializationError` | 110 | — |
| `ContentDeploymentPackageNotReadyError` | 111 | — |
| `ContentUploadFailed` | 112 | — |
| `ContentTooManyResults` | 113 | — |
| `ContentInvalidState` | 115 | — |
| `ContentNavigationParentNotFound` | 116 | — |
| `ContentNavigationParentUpdateError` | 117 | — |
| `DeploymentPackageNotEditable` | 118 | — |
| `ContentValidationError` | 119 | — |
| `ContentPropertiesValidationError` | 120 | — |
| `ContentTypeNotFound` | 121 | — |
| `DeploymentPackageNotFound` | 122 | — |
| `ContentSearchInvalidParameters` | 123 | — |
| `ContentItemPropertyAggregationError` | 124 | — |
| `DeploymentPackageFileNotFound` | 125 | — |
| `ContentPerforceFileHistoryNotFound` | 126 | — |
| `ContentAssetZipCreationFailure` | 127 | — |
| `ContentAssetZipCreationBusy` | 128 | — |
| `ContentProjectNotFound` | 129 | — |
| `ContentFolderNotFound` | 130 | — |
| `ContentPackagesInconsistent` | 131 | — |
| `ContentPackagesInvalidState` | 132 | — |
| `ContentPackagesInconsistentType` | 133 | — |
| `ContentCannotDeletePackage` | 134 | — |
| `ContentLockedForChanges` | 135 | — |
| `ContentFileUploadFailed` | 136 | — |
| `ContentNotReviewed` | 137 | — |
| `ContentPermissionDenied` | 138 | — |
| `ContentInvalidExternalUrl` | 139 | — |
| `ContentExternalFileCannotBeImportedLocally` | 140 | — |
| `ContentTagSaveFailure` | 141 | — |
| `ContentPerforceUnmatchedFileError` | 142 | — |
| `ContentPerforceChangelistResultNotFound` | 143 | — |
| `ContentPerforceChangelistFileItemsNotFound` | 144 | — |
| `ContentPerforceInvalidRevisionError` | 145 | — |
| `ContentUnloadedSaveResult` | 146 | — |
| `ContentPropertyInvalidNumber` | 147 | — |
| `ContentPropertyInvalidUrl` | 148 | — |
| `ContentPropertyInvalidDate` | 149 | — |
| `ContentPropertyInvalidSet` | 150 | — |
| `ContentPropertyCannotDeserialize` | 151 | — |
| `ContentRegexValidationFailOnProperty` | 152 | — |
| `ContentMaxLengthFailOnProperty` | 153 | — |
| `ContentPropertyUnexpectedDeserializationError` | 154 | — |
| `ContentPropertyRequired` | 155 | — |
| `ContentCannotCreateFile` | 156 | — |
| `ContentInvalidMigrationFile` | 157 | — |
| `ContentMigrationAlteringProcessedItem` | 158 | — |
| `ContentPropertyDefinitionNotFound` | 159 | — |
| `ContentReviewDataChanged` | 160 | — |
| `ContentRollbackRevisionNotInPackage` | 161 | — |
| `ContentItemNotBasedOnLatestRevision` | 162 | — |
| `ContentUnauthorized` | 163 | — |
| `ContentCannotCreateDeploymentPackage` | 164 | — |
| `ContentUserNotFound` | 165 | — |
| `ContentLocalePermissionDenied` | 166 | — |
| `ContentInvalidLinkToInternalEnvironment` | 167 | — |
| `ContentInvalidBlacklistedContent` | 168 | — |
| `ContentMacroMalformedNoContentId` | 169 | — |
| `ContentMacroMalformedNoTemplateType` | 170 | — |
| `ContentIllegalBNetMembershipId` | 171 | — |
| `ContentLocaleDidNotMatchExpected` | 172 | — |
| `ContentBabelCallFailed` | 173 | — |
| `ContentEnglishPostLiveForbidden` | 174 | — |
| `ContentLocaleEditPermissionDenied` | 175 | — |
| `ContentStackUnknownError` | 176 | — |
| `ContentStackNotFound` | 177 | — |
| `ContentStackRateLimited` | 178 | — |
| `ContentStackTimeout` | 179 | — |
| `ContentStackServiceError` | 180 | — |
| `ContentStackDeserializationFailure` | 181 | — |
| `UserNonUniqueName` | 200 | — |
| `UserManualLinkingStepRequired` | 201 | — |
| `UserCreateUnknownSqlResult` | 202 | — |
| `UserCreateUnknownSqlException` | 203 | — |
| `UserMalformedMembershipId` | 204 | — |
| `UserCannotFindRequestedUser` | 205 | — |
| `UserCannotLoadAccountCredentialLinkInfo` | 206 | — |
| `UserInvalidMobileAppType` | 207 | — |
| `UserMissingMobilePairingInfo` | 208 | — |
| `UserCannotGenerateMobileKeyWhileUsingMobileCredential` | 209 | — |
| `UserGenerateMobileKeyExistingSlotCollision` | 210 | — |
| `UserDisplayNameMissingOrInvalid` | 211 | — |
| `UserCannotLoadAccountProfileData` | 212 | — |
| `UserCannotSaveUserProfileData` | 213 | — |
| `UserEmailMissingOrInvalid` | 214 | — |
| `UserTermsOfUseRequired` | 215 | — |
| `UserCannotCreateNewAccountWhileLoggedIn` | 216 | — |
| `UserCannotResolveCentralAccount` | 217 | — |
| `UserInvalidAvatar` | 218 | — |
| `UserMissingCreatedUserResult` | 219 | — |
| `UserCannotChangeUniqueNameYet` | 220 | — |
| `UserCannotChangeDisplayNameYet` | 221 | — |
| `UserCannotChangeEmail` | 222 | — |
| `UserUniqueNameMustStartWithLetter` | 223 | — |
| `UserNoLinkedAccountsSupportFriendListings` | 224 | — |
| `UserAcknowledgmentTableFull` | 225 | — |
| `UserCreationDestinyMembershipRequired` | 226 | — |
| `UserFriendsTokenNeedsRefresh` | 227 | — |
| `UserEmailValidationUnknown` | 228 | — |
| `UserEmailValidationLimit` | 229 | — |
| `TransactionEmailSendFailure` | 230 | — |
| `MailHookPermissionFailure` | 231 | — |
| `MailServiceRateLimit` | 232 | — |
| `UserEmailMustBeVerified` | 233 | — |
| `UserMustAllowCustomerServiceEmails` | 234 | — |
| `NonTransactionalEmailSendFailure` | 235 | — |
| `UnknownErrorSettingGlobalDisplayName` | 236 | — |
| `DuplicateGlobalDisplayName` | 237 | — |
| `ErrorRunningNameValidationChecks` | 238 | — |
| `ErrorDatabaseGlobalName` | 239 | — |
| `ErrorNoAvailableNameChanges` | 240 | — |
| `ErrorNameAlreadySetToInput` | 241 | — |
| `UserDisplayNameLessThanMinLength` | 242 | — |
| `UserDisplayNameGreaterThanMaxLength` | 243 | — |
| `UserDisplayNameContainsUnacceptableOrInvalidContent` | 244 | — |
| `EmailValidationOffline` | 245 | — |
| `EmailValidationFailOldCode` | 246 | — |
| `EmailValidationFailBadLink` | 247 | — |
| `EmailUnsubscribeFail` | 248 | — |
| `EmailUnsubscribeFailNew` | 249 | — |
| `MessagingUnknownError` | 300 | — |
| `MessagingSelfError` | 301 | — |
| `MessagingSendThrottle` | 302 | — |
| `MessagingNoBody` | 303 | — |
| `MessagingTooManyUsers` | 304 | — |
| `MessagingCanNotLeaveConversation` | 305 | — |
| `MessagingUnableToSend` | 306 | — |
| `MessagingDeletedUserForbidden` | 307 | — |
| `MessagingCannotDeleteExternalConversation` | 308 | — |
| `MessagingGroupChatDisabled` | 309 | — |
| `MessagingMustIncludeSelfInPrivateMessage` | 310 | — |
| `MessagingSenderIsBanned` | 311 | — |
| `MessagingGroupOptionalChatExceededMaximum` | 312 | — |
| `PrivateMessagingRequiresDestinyMembership` | 313 | — |
| `MessagingSendDailyThrottle` | 314 | — |
| `AddSurveyAnswersUnknownSqlException` | 400 | — |
| `ForumBodyCannotBeEmpty` | 500 | — |
| `ForumSubjectCannotBeEmptyOnTopicPost` | 501 | — |
| `ForumCannotLocateParentPost` | 502 | — |
| `ForumThreadLockedForReplies` | 503 | — |
| `ForumUnknownSqlResultDuringCreatePost` | 504 | — |
| `ForumUnknownTagCreationError` | 505 | — |
| `ForumUnknownSqlResultDuringTagItem` | 506 | — |
| `ForumUnknownExceptionCreatePost` | 507 | — |
| `ForumQuestionMustBeTopicPost` | 508 | — |
| `ForumExceptionDuringTagSearch` | 509 | — |
| `ForumExceptionDuringTopicRetrieval` | 510 | — |
| `ForumAliasedTagError` | 511 | — |
| `ForumCannotLocateThread` | 512 | — |
| `ForumUnknownExceptionEditPost` | 513 | — |
| `ForumCannotLocatePost` | 514 | — |
| `ForumUnknownExceptionGetOrCreateTags` | 515 | — |
| `ForumEditPermissionDenied` | 516 | — |
| `ForumUnknownSqlResultDuringTagIdRetrieval` | 517 | — |
| `ForumCannotGetRating` | 518 | — |
| `ForumUnknownExceptionGetRating` | 519 | — |
| `ForumRatingsAccessError` | 520 | — |
| `ForumRelatedPostAccessError` | 521 | — |
| `ForumLatestReplyAccessError` | 522 | — |
| `ForumUserStatusAccessError` | 523 | — |
| `ForumAuthorAccessError` | 524 | — |
| `ForumGroupAccessError` | 525 | — |
| `ForumUrlExpectedButMissing` | 526 | — |
| `ForumRepliesCannotBeEmpty` | 527 | — |
| `ForumRepliesCannotBeInDifferentGroups` | 528 | — |
| `ForumSubTopicCannotBeCreatedAtThisThreadLevel` | 529 | — |
| `ForumCannotCreateContentTopic` | 530 | — |
| `ForumTopicDoesNotExist` | 531 | — |
| `ForumContentCommentsNotAllowed` | 532 | — |
| `ForumUnknownSqlResultDuringEditPost` | 533 | — |
| `ForumUnknownSqlResultDuringGetPost` | 534 | — |
| `ForumPostValidationBadUrl` | 535 | — |
| `ForumBodyTooLong` | 536 | — |
| `ForumSubjectTooLong` | 537 | — |
| `ForumAnnouncementNotAllowed` | 538 | — |
| `ForumCannotShareOwnPost` | 539 | — |
| `ForumEditNoOp` | 540 | — |
| `ForumUnknownDatabaseErrorDuringGetPost` | 541 | — |
| `ForumExceeedMaximumRowLimit` | 542 | — |
| `ForumCannotSharePrivatePost` | 543 | — |
| `ForumCannotCrossPostBetweenGroups` | 544 | — |
| `ForumIncompatibleCategories` | 555 | — |
| `ForumCannotUseTheseCategoriesOnNonTopicPost` | 556 | — |
| `ForumCanOnlyDeleteTopics` | 557 | — |
| `ForumDeleteSqlException` | 558 | — |
| `ForumDeleteSqlUnknownResult` | 559 | — |
| `ForumTooManyTags` | 560 | — |
| `ForumCanOnlyRateTopics` | 561 | — |
| `ForumBannedPostsCannotBeEdited` | 562 | — |
| `ForumThreadRootIsBanned` | 563 | — |
| `ForumCannotUseOfficialTagCategoryAsTag` | 564 | — |
| `ForumAnswerCannotBeMadeOnCreatePost` | 565 | — |
| `ForumAnswerCannotBeMadeOnEditPost` | 566 | — |
| `ForumAnswerPostIdIsNotADirectReplyOfQuestion` | 567 | — |
| `ForumAnswerTopicIdIsNotAQuestion` | 568 | — |
| `ForumUnknownExceptionDuringMarkAnswer` | 569 | — |
| `ForumUnknownSqlResultDuringMarkAnswer` | 570 | — |
| `ForumCannotRateYourOwnPosts` | 571 | — |
| `ForumPollsMustBeTheFirstPostInTopic` | 572 | — |
| `ForumInvalidPollInput` | 573 | — |
| `ForumGroupAdminEditNonMember` | 574 | — |
| `ForumCannotEditModeratorEditedPost` | 575 | — |
| `ForumRequiresDestinyMembership` | 576 | — |
| `ForumUnexpectedError` | 577 | — |
| `ForumAgeLock` | 578 | — |
| `ForumMaxPages` | 579 | — |
| `ForumMaxPagesOldestFirst` | 580 | — |
| `ForumCannotApplyForumIdWithoutTags` | 581 | — |
| `ForumCannotApplyForumIdToNonTopics` | 582 | — |
| `ForumCannotDownvoteCommunityCreations` | 583 | — |
| `ForumTopicsMustHaveOfficialCategory` | 584 | — |
| `ForumRecruitmentTopicMalformed` | 585 | — |
| `ForumRecruitmentTopicNotFound` | 586 | — |
| `ForumRecruitmentTopicNoSlotsRemaining` | 587 | — |
| `ForumRecruitmentTopicKickBan` | 588 | — |
| `ForumRecruitmentTopicRequirementsNotMet` | 589 | — |
| `ForumRecruitmentTopicNoPlayers` | 590 | — |
| `ForumRecruitmentApproveFailMessageBan` | 591 | — |
| `ForumRecruitmentGlobalBan` | 592 | — |
| `ForumUserBannedFromThisTopic` | 593 | — |
| `ForumRecruitmentFireteamMembersOnly` | 594 | — |
| `ForumRequiresDestiny2Progress` | 595 | — |
| `ForumRequiresDestiny2EntitlementPurchase` | 596 | — |
| `GroupMembershipApplicationAlreadyResolved` | 601 | — |
| `GroupMembershipAlreadyApplied` | 602 | — |
| `GroupMembershipInsufficientPrivileges` | 603 | — |
| `GroupIdNotReturnedFromCreation` | 604 | — |
| `GroupSearchInvalidParameters` | 605 | — |
| `GroupMembershipPendingApplicationNotFound` | 606 | — |
| `GroupInvalidId` | 607 | — |
| `GroupInvalidMembershipId` | 608 | — |
| `GroupInvalidMembershipType` | 609 | — |
| `GroupMissingTags` | 610 | — |
| `GroupMembershipNotFound` | 611 | — |
| `GroupInvalidRating` | 612 | — |
| `GroupUserFollowingAccessError` | 613 | — |
| `GroupUserMembershipAccessError` | 614 | — |
| `GroupCreatorAccessError` | 615 | — |
| `GroupAdminAccessError` | 616 | — |
| `GroupPrivatePostNotViewable` | 617 | — |
| `GroupMembershipNotLoggedIn` | 618 | — |
| `GroupNotDeleted` | 619 | — |
| `GroupUnknownErrorUndeletingGroup` | 620 | — |
| `GroupDeleted` | 621 | — |
| `GroupNotFound` | 622 | — |
| `GroupMemberBanned` | 623 | — |
| `GroupMembershipClosed` | 624 | — |
| `GroupPrivatePostOverrideError` | 625 | — |
| `GroupNameTaken` | 626 | — |
| `GroupDeletionGracePeriodExpired` | 627 | — |
| `GroupCannotCheckBanStatus` | 628 | — |
| `GroupMaximumMembershipCountReached` | 629 | — |
| `NoDestinyAccountForClanPlatform` | 630 | — |
| `AlreadyRequestingMembershipForClanPlatform` | 631 | — |
| `AlreadyClanMemberOnPlatform` | 632 | — |
| `GroupJoinedCannotSetClanName` | 633 | — |
| `GroupLeftCannotClearClanName` | 634 | — |
| `GroupRelationshipRequestPending` | 635 | — |
| `GroupRelationshipRequestBlocked` | 636 | — |
| `GroupRelationshipRequestNotFound` | 637 | — |
| `GroupRelationshipBlockNotFound` | 638 | — |
| `GroupRelationshipNotFound` | 639 | — |
| `GroupAlreadyAllied` | 641 | — |
| `GroupAlreadyMember` | 642 | — |
| `GroupRelationshipAlreadyExists` | 643 | — |
| `InvalidGroupTypesForRelationshipRequest` | 644 | — |
| `GroupAtMaximumAlliances` | 646 | — |
| `GroupCannotSetClanOnlySettings` | 647 | — |
| `ClanCannotSetTwoDefaultPostTypes` | 648 | — |
| `GroupMemberInvalidMemberType` | 649 | — |
| `GroupInvalidPlatformType` | 650 | — |
| `GroupMemberInvalidSort` | 651 | — |
| `GroupInvalidResolveState` | 652 | — |
| `ClanAlreadyEnabledForPlatform` | 653 | — |
| `ClanNotEnabledForPlatform` | 654 | — |
| `ClanEnabledButCouldNotJoinNoAccount` | 655 | — |
| `ClanEnabledButCouldNotJoinAlreadyMember` | 656 | — |
| `ClanCannotJoinNoCredential` | 657 | — |
| `NoClanMembershipForPlatform` | 658 | — |
| `GroupToGroupFollowLimitReached` | 659 | — |
| `ChildGroupAlreadyInAlliance` | 660 | — |
| `OwnerGroupAlreadyInAlliance` | 661 | — |
| `AllianceOwnerCannotJoinAlliance` | 662 | — |
| `GroupNotInAlliance` | 663 | — |
| `ChildGroupCannotInviteToAlliance` | 664 | — |
| `GroupToGroupAlreadyFollowed` | 665 | — |
| `GroupToGroupNotFollowing` | 666 | — |
| `ClanMaximumMembershipReached` | 667 | — |
| `ClanNameNotValid` | 668 | — |
| `ClanNameNotValidError` | 669 | — |
| `AllianceOwnerNotDefined` | 670 | — |
| `AllianceChildNotDefined` | 671 | — |
| `ClanCultureIllegalCharacters` | 672 | — |
| `ClanTagIllegalCharacters` | 673 | — |
| `ClanRequiresInvitation` | 674 | — |
| `ClanMembershipClosed` | 675 | — |
| `ClanInviteAlreadyMember` | 676 | — |
| `GroupInviteAlreadyMember` | 677 | — |
| `GroupJoinApprovalRequired` | 678 | — |
| `ClanTagRequired` | 679 | — |
| `GroupNameCannotStartOrEndWithWhiteSpace` | 680 | — |
| `ClanCallsignCannotStartOrEndWithWhiteSpace` | 681 | — |
| `ClanMigrationFailed` | 682 | — |
| `ClanNotEnabledAlreadyMemberOfAnotherClan` | 683 | — |
| `GroupModerationNotPermittedOnNonMembers` | 684 | — |
| `ClanCreationInWorldServerFailed` | 685 | — |
| `ClanNotFound` | 686 | — |
| `ClanMembershipLevelDoesNotPermitThatAction` | 687 | — |
| `ClanMemberNotFound` | 688 | — |
| `ClanMissingMembershipApprovers` | 689 | — |
| `ClanInWrongStateForRequestedAction` | 690 | — |
| `ClanNameAlreadyUsed` | 691 | — |
| `ClanTooFewMembers` | 692 | — |
| `ClanInfoCannotBeWhitespace` | 693 | — |
| `GroupCultureThrottle` | 694 | — |
| `ClanTargetDisallowsInvites` | 695 | — |
| `ClanInvalidOperation` | 696 | — |
| `ClanFounderCannotLeaveWithoutAbdication` | 697 | — |
| `ClanNameReserved` | 698 | — |
| `ClanApplicantInClanSoNowInvited` | 699 | — |
| `ActivitiesUnknownException` | 701 | — |
| `ActivitiesParameterNull` | 702 | — |
| `ActivityCountsDiabled` | 703 | — |
| `ActivitySearchInvalidParameters` | 704 | — |
| `ActivityPermissionDenied` | 705 | — |
| `ShareAlreadyShared` | 706 | — |
| `ActivityLoggingDisabled` | 707 | — |
| `ClanRequiresExistingDestinyAccount` | 750 | — |
| `ClanNameRestricted` | 751 | — |
| `ClanCreationBan` | 752 | — |
| `ClanCreationTenureRequirementsNotMet` | 753 | — |
| `ClanFieldContainsReservedTerms` | 754 | — |
| `ClanFieldContainsInappropriateContent` | 755 | — |
| `ItemAlreadyFollowed` | 801 | — |
| `ItemNotFollowed` | 802 | — |
| `CannotFollowSelf` | 803 | — |
| `GroupFollowLimitExceeded` | 804 | — |
| `TagFollowLimitExceeded` | 805 | — |
| `UserFollowLimitExceeded` | 806 | — |
| `FollowUnsupportedEntityType` | 807 | — |
| `NoValidTagsInList` | 900 | — |
| `BelowMinimumSuggestionLength` | 901 | — |
| `CannotGetSuggestionsOnMultipleTagsSimultaneously` | 902 | — |
| `NotAValidPartialTag` | 903 | — |
| `TagSuggestionsUnknownSqlResult` | 904 | — |
| `TagsUnableToLoadPopularTagsFromDatabase` | 905 | — |
| `TagInvalid` | 906 | — |
| `TagNotFound` | 907 | — |
| `SingleTagExpected` | 908 | — |
| `TagsExceededMaximumPerItem` | 909 | — |
| `IgnoreInvalidParameters` | 1000 | — |
| `IgnoreSqlException` | 1001 | — |
| `IgnoreErrorRetrievingGroupPermissions` | 1002 | — |
| `IgnoreErrorInsufficientPermission` | 1003 | — |
| `IgnoreErrorRetrievingItem` | 1004 | — |
| `IgnoreCannotIgnoreSelf` | 1005 | — |
| `IgnoreIllegalType` | 1006 | — |
| `IgnoreNotFound` | 1007 | — |
| `IgnoreUserGloballyIgnored` | 1008 | — |
| `IgnoreUserIgnored` | 1009 | — |
| `TargetUserIgnored` | 1010 | — |
| `NotificationSettingInvalid` | 1100 | — |
| `PsnApiExpiredAccessToken` | 1204 | — |
| `PSNExForbidden` | 1205 | — |
| `PSNExSystemDisabled` | 1218 | — |
| `PsnApiErrorCodeUnknown` | 1223 | — |
| `PsnApiErrorWebException` | 1224 | — |
| `PsnApiBadRequest` | 1225 | — |
| `PsnApiAccessTokenRequired` | 1226 | — |
| `PsnApiInvalidAccessToken` | 1227 | — |
| `PsnApiBannedUser` | 1229 | — |
| `PsnApiAccountUpgradeRequired` | 1230 | — |
| `PsnApiServiceTemporarilyUnavailable` | 1231 | — |
| `PsnApiServerBusy` | 1232 | — |
| `PsnApiUnderMaintenance` | 1233 | — |
| `PsnApiProfileUserNotFound` | 1234 | — |
| `PsnApiProfilePrivacyRestriction` | 1235 | — |
| `PsnApiProfileUnderMaintenance` | 1236 | — |
| `PsnApiAccountAttributeMissing` | 1237 | — |
| `PsnApiNoPermission` | 1238 | — |
| `PsnApiTargetUserBlocked` | 1239 | — |
| `PsnApiJwksMissing` | 1240 | — |
| `PsnApiJwtMalformedHeader` | 1241 | — |
| `PsnApiJwtMalformedPayload` | 1242 | — |
| `XblExSystemDisabled` | 1300 | — |
| `XblExUnknownError` | 1301 | — |
| `XblApiErrorWebException` | 1302 | — |
| `XblStsTokenInvalid` | 1303 | — |
| `XblStsMissingToken` | 1304 | — |
| `XblStsExpiredToken` | 1305 | — |
| `XblAccessToTheSandboxDenied` | 1306 | — |
| `XblMsaResponseMissing` | 1307 | — |
| `XblMsaAccessTokenExpired` | 1308 | — |
| `XblMsaInvalidRequest` | 1309 | — |
| `XblMsaFriendsRequireSignIn` | 1310 | — |
| `XblUserActionRequired` | 1311 | — |
| `XblParentalControls` | 1312 | — |
| `XblDeveloperAccount` | 1313 | — |
| `XblUserTokenExpired` | 1314 | — |
| `XblUserTokenInvalid` | 1315 | — |
| `XblOffline` | 1316 | — |
| `XblUnknownErrorCode` | 1317 | — |
| `XblMsaInvalidGrant` | 1318 | — |
| `ReportNotYetResolved` | 1400 | — |
| `ReportOverturnDoesNotChangeDecision` | 1401 | — |
| `ReportNotFound` | 1402 | — |
| `ReportAlreadyReported` | 1403 | — |
| `ReportInvalidResolution` | 1404 | — |
| `ReportNotAssignedToYou` | 1405 | — |
| `LegacyGameStatsSystemDisabled` | 1500 | — |
| `LegacyGameStatsUnknownError` | 1501 | — |
| `LegacyGameStatsMalformedSneakerNetCode` | 1502 | — |
| `DestinyAccountAcquisitionFailure` | 1600 | — |
| `DestinyAccountNotFound` | 1601 | — |
| `DestinyBuildStatsDatabaseError` | 1602 | — |
| `DestinyCharacterStatsDatabaseError` | 1603 | — |
| `DestinyPvPStatsDatabaseError` | 1604 | — |
| `DestinyPvEStatsDatabaseError` | 1605 | — |
| `DestinyGrimoireStatsDatabaseError` | 1606 | — |
| `DestinyStatsParameterMembershipTypeParseError` | 1607 | — |
| `DestinyStatsParameterMembershipIdParseError` | 1608 | — |
| `DestinyStatsParameterRangeParseError` | 1609 | — |
| `DestinyStringItemHashNotFound` | 1610 | — |
| `DestinyStringSetNotFound` | 1611 | — |
| `DestinyContentLookupNotFoundForKey` | 1612 | — |
| `DestinyContentItemNotFound` | 1613 | — |
| `DestinyContentSectionNotFound` | 1614 | — |
| `DestinyContentPropertyNotFound` | 1615 | — |
| `DestinyContentConfigNotFound` | 1616 | — |
| `DestinyContentPropertyBucketValueNotFound` | 1617 | — |
| `DestinyUnexpectedError` | 1618 | — |
| `DestinyInvalidAction` | 1619 | — |
| `DestinyCharacterNotFound` | 1620 | — |
| `DestinyInvalidFlag` | 1621 | — |
| `DestinyInvalidRequest` | 1622 | — |
| `DestinyItemNotFound` | 1623 | — |
| `DestinyInvalidCustomizationChoices` | 1624 | — |
| `DestinyVendorItemNotFound` | 1625 | — |
| `DestinyInternalError` | 1626 | — |
| `DestinyVendorNotFound` | 1627 | — |
| `DestinyRecentActivitiesDatabaseError` | 1628 | — |
| `DestinyItemBucketNotFound` | 1629 | — |
| `DestinyInvalidMembershipType` | 1630 | — |
| `DestinyVersionIncompatibility` | 1631 | — |
| `DestinyItemAlreadyInInventory` | 1632 | — |
| `DestinyBucketNotFound` | 1633 | — |
| `DestinyCharacterNotInTower` | 1634 | Note: This is one of those holdovers from Destiny 1. We didn't change the enum because I am lazy, but in Destiny 2 this would read "DestinyCharacterNotInSocialSpace" |
| `DestinyCharacterNotLoggedIn` | 1635 | — |
| `DestinyDefinitionsNotLoaded` | 1636 | — |
| `DestinyInventoryFull` | 1637 | — |
| `DestinyItemFailedLevelCheck` | 1638 | — |
| `DestinyItemFailedUnlockCheck` | 1639 | — |
| `DestinyItemUnequippable` | 1640 | — |
| `DestinyItemUniqueEquipRestricted` | 1641 | — |
| `DestinyNoRoomInDestination` | 1642 | — |
| `DestinyServiceFailure` | 1643 | — |
| `DestinyServiceRetired` | 1644 | — |
| `DestinyTransferFailed` | 1645 | — |
| `DestinyTransferNotFoundForSourceBucket` | 1646 | — |
| `DestinyUnexpectedResultInVendorTransferCheck` | 1647 | — |
| `DestinyUniquenessViolation` | 1648 | — |
| `DestinyErrorDeserializationFailure` | 1649 | — |
| `DestinyValidAccountTicketRequired` | 1650 | — |
| `DestinyShardRelayClientTimeout` | 1651 | — |
| `DestinyShardRelayProxyTimeout` | 1652 | — |
| `DestinyPGCRNotFound` | 1653 | — |
| `DestinyAccountMustBeOffline` | 1654 | — |
| `DestinyCanOnlyEquipInGame` | 1655 | — |
| `DestinyCannotPerformActionOnEquippedItem` | 1656 | — |
| `DestinyQuestAlreadyCompleted` | 1657 | — |
| `DestinyQuestAlreadyTracked` | 1658 | — |
| `DestinyTrackableQuestsFull` | 1659 | — |
| `DestinyItemNotTransferrable` | 1660 | — |
| `DestinyVendorPurchaseNotAllowed` | 1661 | — |
| `DestinyContentVersionMismatch` | 1662 | — |
| `DestinyItemActionForbidden` | 1663 | — |
| `DestinyRefundInvalid` | 1664 | — |
| `DestinyPrivacyRestriction` | 1665 | — |
| `DestinyActionInsufficientPrivileges` | 1666 | — |
| `DestinyInvalidClaimException` | 1667 | — |
| `DestinyLegacyPlatformRestricted` | 1668 | — |
| `DestinyLegacyPlatformInUse` | 1669 | — |
| `DestinyLegacyPlatformInaccessible` | 1670 | — |
| `DestinyCannotPerformActionAtThisLocation` | 1671 | — |
| `DestinyThrottledByGameServer` | 1672 | — |
| `DestinyItemNotTransferrableHasSideEffects` | 1673 | — |
| `DestinyItemLocked` | 1674 | — |
| `DestinyCannotAffordMaterialRequirements` | 1675 | — |
| `DestinyFailedPlugInsertionRules` | 1676 | — |
| `DestinySocketNotFound` | 1677 | — |
| `DestinySocketActionNotAllowed` | 1678 | — |
| `DestinySocketAlreadyHasPlug` | 1679 | — |
| `DestinyPlugItemNotAvailable` | 1680 | — |
| `DestinyCharacterLoggedInNotAllowed` | 1681 | — |
| `DestinyPublicAccountNotAccessible` | 1682 | — |
| `DestinyClaimsItemAlreadyClaimed` | 1683 | — |
| `DestinyClaimsNoInventorySpace` | 1684 | — |
| `DestinyClaimsRequiredLevelNotMet` | 1685 | — |
| `DestinyClaimsInvalidState` | 1686 | — |
| `DestinyNotEnoughRoomForMultipleRewards` | 1687 | — |
| `DestinyDirectBabelClientTimeout` | 1688 | — |
| `FbInvalidRequest` | 1800 | — |
| `FbRedirectMismatch` | 1801 | — |
| `FbAccessDenied` | 1802 | — |
| `FbUnsupportedResponseType` | 1803 | — |
| `FbInvalidScope` | 1804 | — |
| `FbUnsupportedGrantType` | 1805 | — |
| `FbInvalidGrant` | 1806 | — |
| `InvitationExpired` | 1900 | — |
| `InvitationUnknownType` | 1901 | — |
| `InvitationInvalidResponseStatus` | 1902 | — |
| `InvitationInvalidType` | 1903 | — |
| `InvitationAlreadyPending` | 1904 | — |
| `InvitationInsufficientPermission` | 1905 | — |
| `InvitationInvalidCode` | 1906 | — |
| `InvitationInvalidTargetState` | 1907 | — |
| `InvitationCannotBeReactivated` | 1908 | — |
| `InvitationNoRecipients` | 1910 | — |
| `InvitationGroupCannotSendToSelf` | 1911 | — |
| `InvitationTooManyRecipients` | 1912 | — |
| `InvitationInvalid` | 1913 | — |
| `InvitationNotFound` | 1914 | — |
| `TokenInvalid` | 2000 | — |
| `TokenBadFormat` | 2001 | — |
| `TokenAlreadyClaimed` | 2002 | — |
| `TokenAlreadyClaimedSelf` | 2003 | — |
| `TokenThrottling` | 2004 | — |
| `TokenUnknownRedemptionFailure` | 2005 | — |
| `TokenPurchaseClaimFailedAfterTokenClaimed` | 2006 | — |
| `TokenUserAlreadyOwnsOffer` | 2007 | — |
| `TokenInvalidOfferKey` | 2008 | — |
| `TokenEmailNotValidated` | 2009 | — |
| `TokenProvisioningBadVendorOrOffer` | 2010 | — |
| `TokenPurchaseHistoryUnknownError` | 2011 | — |
| `TokenThrottleStateUnknownError` | 2012 | — |
| `TokenUserAgeNotVerified` | 2013 | — |
| `TokenExceededOfferMaximum` | 2014 | — |
| `TokenNoAvailableUnlocks` | 2015 | — |
| `TokenMarketplaceInvalidPlatform` | 2016 | — |
| `TokenNoMarketplaceCodesFound` | 2017 | — |
| `TokenOfferNotAvailableForRedemption` | 2018 | — |
| `TokenUnlockPartialFailure` | 2019 | — |
| `TokenMarketplaceInvalidRegion` | 2020 | — |
| `TokenOfferExpired` | 2021 | — |
| `RAFExceededMaximumReferrals` | 2022 | — |
| `RAFDuplicateBond` | 2023 | — |
| `RAFNoValidVeteranDestinyMembershipsFound` | 2024 | — |
| `RAFNotAValidVeteranUser` | 2025 | — |
| `RAFCodeAlreadyClaimedOrNotFound` | 2026 | — |
| `RAFMismatchedDestinyMembershipType` | 2027 | — |
| `RAFUnableToAccessPurchaseHistory` | 2028 | — |
| `RAFUnableToCreateBond` | 2029 | — |
| `RAFUnableToFindBond` | 2030 | — |
| `RAFUnableToRemoveBond` | 2031 | — |
| `RAFCannotBondToSelf` | 2032 | — |
| `RAFInvalidPlatform` | 2033 | — |
| `RAFGenerateThrottled` | 2034 | — |
| `RAFUnableToCreateBondVersionMismatch` | 2035 | — |
| `RAFUnableToRemoveBondVersionMismatch` | 2036 | — |
| `RAFRedeemThrottled` | 2037 | — |
| `NoAvailableDiscountCode` | 2038 | — |
| `DiscountAlreadyClaimed` | 2039 | — |
| `DiscountClaimFailure` | 2040 | — |
| `DiscountConfigurationFailure` | 2041 | — |
| `DiscountGenerationFailure` | 2042 | — |
| `DiscountAlreadyExists` | 2043 | — |
| `TokenRequiresCredentialXuid` | 2044 | — |
| `TokenRequiresCredentialPsnid` | 2045 | — |
| `OfferRequired` | 2046 | — |
| `UnknownEververseHistoryError` | 2047 | — |
| `MissingEververseHistoryError` | 2048 | — |
| `BungieRewardEmailStateInvalid` | 2049 | — |
| `BungieRewardNotYetClaimable` | 2050 | — |
| `MissingOfferConfig` | 2051 | — |
| `RAFQuestEntitlementRequiresBnet` | 2052 | — |
| `RAFQuestEntitlementTransportFailure` | 2053 | — |
| `RAFQuestEntitlementUnknownFailure` | 2054 | — |
| `RAFVeteranRewardUnknownFailure` | 2055 | — |
| `RAFTooEarlyToCancelBond` | 2056 | — |
| `LoyaltyRewardAlreadyRedeemed` | 2057 | — |
| `UnclaimedLoyaltyRewardEntryNotFound` | 2058 | — |
| `PartnerOfferPartialFailure` | 2059 | — |
| `PartnerOfferAlreadyClaimed` | 2060 | — |
| `PartnerOfferSkuNotFound` | 2061 | — |
| `PartnerOfferSkuExpired` | 2062 | — |
| `PartnerOfferPermissionFailure` | 2063 | — |
| `PartnerOfferNoDestinyAccount` | 2064 | — |
| `PartnerOfferApplyDataNotFound` | 2065 | — |
| `ApiExceededMaxKeys` | 2100 | — |
| `ApiInvalidOrExpiredKey` | 2101 | — |
| `ApiKeyMissingFromRequest` | 2102 | — |
| `ApplicationDisabled` | 2103 | — |
| `ApplicationExceededMax` | 2104 | — |
| `ApplicationDisallowedByScope` | 2105 | — |
| `AuthorizationCodeInvalid` | 2106 | — |
| `OriginHeaderDoesNotMatchKey` | 2107 | — |
| `AccessNotPermittedByApplicationScope` | 2108 | — |
| `ApplicationNameIsTaken` | 2109 | — |
| `RefreshTokenNotYetValid` | 2110 | — |
| `AccessTokenHasExpired` | 2111 | — |
| `ApplicationTokenFormatNotValid` | 2112 | — |
| `ApplicationNotConfiguredForBungieAuth` | 2113 | — |
| `ApplicationNotConfiguredForOAuth` | 2114 | — |
| `OAuthAccessTokenExpired` | 2115 | — |
| `ApplicationTokenKeyIdDoesNotExist` | 2116 | — |
| `ProvidedTokenNotValidRefreshToken` | 2117 | — |
| `RefreshTokenExpired` | 2118 | — |
| `AuthorizationRecordInvalid` | 2119 | — |
| `TokenPreviouslyRevoked` | 2120 | — |
| `TokenInvalidMembership` | 2121 | — |
| `AuthorizationCodeStale` | 2122 | — |
| `AuthorizationRecordExpired` | 2123 | — |
| `AuthorizationRecordRevoked` | 2124 | — |
| `AuthorizationRecordInactiveApiKey` | 2125 | — |
| `AuthorizationRecordApiKeyMatching` | 2126 | — |
| `PartnershipInvalidType` | 2200 | — |
| `PartnershipValidationError` | 2201 | — |
| `PartnershipValidationTimeout` | 2202 | — |
| `PartnershipAccessFailure` | 2203 | — |
| `PartnershipAccountInvalid` | 2204 | — |
| `PartnershipGetAccountInfoFailure` | 2205 | — |
| `PartnershipDisabled` | 2206 | — |
| `PartnershipAlreadyExists` | 2207 | — |
| `CommunityStreamingUnavailable` | 2300 | — |
| `TwitchNotLinked` | 2500 | — |
| `TwitchAccountNotFound` | 2501 | — |
| `TwitchCouldNotLoadDestinyInfo` | 2502 | — |
| `TwitchCouldNotRegisterUser` | 2503 | — |
| `TwitchCouldNotUnregisterUser` | 2504 | — |
| `TwitchRequiresRelinking` | 2505 | — |
| `TwitchNoPlatformChosen` | 2506 | — |
| `TwitchDropHistoryPermissionFailure` | 2507 | — |
| `TwitchDropsRepairPartialFailure` | 2508 | — |
| `TwitchNotAuthorized` | 2509 | — |
| `TwitchUnknownAuthorizationFailure` | 2510 | — |
| `TrendingCategoryNotFound` | 2600 | — |
| `TrendingEntryTypeNotSupported` | 2601 | — |
| `ReportOffenderNotInPgcr` | 2700 | — |
| `ReportRequestorNotInPgcr` | 2701 | — |
| `ReportSubmissionFailed` | 2702 | — |
| `ReportCannotReportSelf` | 2703 | — |
| `AwaTypeDisabled` | 2800 | — |
| `AwaTooManyPendingRequests` | 2801 | — |
| `AwaTheFeatureRequiresARegisteredDevice` | 2802 | — |
| `AwaRequestWasUnansweredForTooLong` | 2803 | — |
| `AwaWriteRequestMissingOrInvalidToken` | 2804 | — |
| `AwaWriteRequestTokenExpired` | 2805 | — |
| `AwaWriteRequestTokenUsageLimitReached` | 2806 | — |
| `SteamWebApiError` | 2900 | — |
| `SteamWebNullResponseError` | 2901 | — |
| `SteamAccountRequired` | 2902 | — |
| `SteamNotAuthorized` | 2903 | — |
| `ClanFireteamNotFound` | 3000 | — |
| `ClanFireteamAddNoAlternatesForImmediate` | 3001 | — |
| `ClanFireteamFull` | 3002 | — |
| `ClanFireteamAltFull` | 3003 | — |
| `ClanFireteamBlocked` | 3004 | — |
| `ClanFireteamPlayerEntryNotFound` | 3005 | — |
| `ClanFireteamPermissions` | 3006 | — |
| `ClanFireteamInvalidPlatform` | 3007 | — |
| `ClanFireteamCannotAdjustSlotCount` | 3008 | — |
| `ClanFireteamInvalidPlayerPlatform` | 3009 | — |
| `ClanFireteamNotReadyForInvitesNotEnoughPlayers` | 3010 | — |
| `ClanFireteamGameInvitesNotSupportForPlatform` | 3011 | — |
| `ClanFireteamPlatformInvitePreqFailure` | 3012 | — |
| `ClanFireteamInvalidAuthContext` | 3013 | — |
| `ClanFireteamInvalidAuthProviderPsn` | 3014 | — |
| `ClanFireteamPs4SessionFull` | 3015 | — |
| `ClanFireteamInvalidAuthToken` | 3016 | — |
| `ClanFireteamScheduledFireteamsDisabled` | 3017 | — |
| `ClanFireteamNotReadyForInvitesNotScheduledYet` | 3018 | — |
| `ClanFireteamNotReadyForInvitesClosed` | 3019 | — |
| `ClanFireteamScheduledFireteamsRequireAdminPermissions` | 3020 | — |
| `ClanFireteamNonPublicMustHaveClan` | 3021 | — |
| `ClanFireteamPublicCreationRestriction` | 3022 | — |
| `ClanFireteamAlreadyJoined` | 3023 | — |
| `ClanFireteamScheduledFireteamsRange` | 3024 | — |
| `ClanFireteamPublicCreationRestrictionExtended` | 3025 | — |
| `ClanFireteamExpired` | 3026 | — |
| `ClanFireteamInvalidAuthProvider` | 3027 | — |
| `ClanFireteamInvalidAuthProviderXuid` | 3028 | — |
| `ClanFireteamThrottle` | 3029 | — |
| `ClanFireteamTooManyOpenScheduledFireteams` | 3030 | — |
| `ClanFireteamCannotReopenScheduledFireteams` | 3031 | — |
| `ClanFireteamJoinNoAccountSpecified` | 3032 | — |
| `ClanFireteamMinDestiny2ProgressForCreation` | 3033 | — |
| `ClanFireteamMinDestiny2ProgressForJoin` | 3034 | — |
| `ClanFireteamSMSOrPurchaseRequiredCreate` | 3035 | — |
| `ClanFireteamPurchaseRequiredCreate` | 3036 | — |
| `ClanFireteamSMSOrPurchaseRequiredJoin` | 3037 | — |
| `ClanFireteamPurchaseRequiredJoin` | 3038 | — |
| `FireteamFinderInvalidMembershipType` | 3100 | — |
| `FireteamFinderInvalidMembershipId` | 3101 | — |
| `FireteamFinderInvalidCharacterId` | 3102 | — |
| `FireteamFinderInvalidListingOptions` | 3103 | — |
| `FireteamFinderInvalidRequestData` | 3104 | — |
| `FireteamFinderListingApplicationFailed` | 3105 | — |
| `FireteamFinderListingAutoJoinFailed` | 3106 | — |
| `FireteamFinderPlayerApplicationsParsingFailed` | 3107 | — |
| `FireteamFinderJoinLobbyHostFailed` | 3108 | — |
| `FireteamFinderPlayerNotInGame` | 3109 | — |
| `FireteamFinderActivationFailed` | 3110 | — |
| `FireteamFinderApplicationNotFound` | 3111 | — |
| `FireteamFinderUserAlreadyAppliedToListing` | 3112 | — |
| `FireteamFinderApplicationClosedForUpdates` | 3113 | — |
| `FireteamFinderListingAtMaxOpenApplicationsLimit` | 3114 | — |
| `FireteamFinderUserNotInApplication` | 3115 | — |
| `FireteamFinderApplicationUserAlreadyListingOwner` | 3116 | — |
| `FireteamFinderOfferNotFound` | 3117 | — |
| `FireteamFinderOfferClosedForUpdates` | 3118 | — |
| `FireteamFinderOfferUserNotTarget` | 3119 | — |
| `FireteamFinderLobbyNotFound` | 3120 | — |
| `FireteamFinderListingNotFound` | 3121 | — |
| `FireteamFinderLobbyFull` | 3122 | — |
| `FireteamFinderUserNotListingOwner` | 3123 | — |
| `FireteamFinderUserNotLobbyOwner` | 3124 | — |
| `FireteamFinderLobbyClosedForUpdates` | 3125 | — |
| `FireteamFinderUserNotInLobby` | 3126 | — |
| `FireteamFinderDisabledSettingsValue` | 3127 | — |
| `FireteamFinderOwnerInActiveLobby` | 3128 | — |
| `FireteamFinderApplicationClosedToOfflinePlayers` | 3129 | — |
| `FireteamFinderUserNotApplicationOwner` | 3130 | — |
| `FireteamFinderInviteValidationFailed` | 3131 | — |
| `FireteamFinderOwnerNotInGame` | 3132 | — |
| `FireteamFinderPlayerAtMaxLobbyLimit` | 3133 | — |
| `FireteamFinderLobbyTooFarInTheFuture` | 3134 | — |
| `FireteamFinderApplicantNotInGame` | 3135 | — |
| `FireteamFinderResponseUndefined` | 3150 | — |
| `FireteamFinderResponseMoved` | 3151 | — |
| `FireteamFinderResponseLoggingIn` | 3152 | — |
| `FireteamFinderResponseBadRequest` | 3153 | — |
| `FireteamFinderResponseUnauthorized` | 3154 | — |
| `FireteamFinderResponseForbidden` | 3155 | — |
| `FireteamFinderResponseNotFound` | 3156 | — |
| `FireteamFinderInternalServerError` | 3157 | — |
| `FireteamFinderServiceUnavailable` | 3158 | — |
| `FireteamFinderInternalServerErrorNonFatal` | 3159 | — |
| `CrossSaveOverriddenAccountNotFound` | 3200 | — |
| `CrossSaveTooManyOverriddenPlatforms` | 3201 | — |
| `CrossSaveNoOverriddenPlatforms` | 3202 | — |
| `CrossSavePrimaryAccountNotFound` | 3203 | — |
| `CrossSaveRequestInvalid` | 3204 | — |
| `CrossSaveBungieAccountValidationFailure` | 3206 | — |
| `CrossSaveOverriddenPlatformNotAllowed` | 3207 | — |
| `CrossSaveThresholdExceeded` | 3208 | — |
| `CrossSaveIncompatibleMembershipType` | 3209 | — |
| `CrossSaveCouldNotFindLinkedAccountForMembershipType` | 3210 | — |
| `CrossSaveCouldNotCreateDestinyProfileForMembershipType` | 3211 | — |
| `CrossSaveErrorCreatingDestinyProfileForMembershipType` | 3212 | — |
| `CrossSaveCannotOverrideSelf` | 3213 | — |
| `CrossSaveRecentSilverPurchase` | 3214 | — |
| `CrossSaveSilverBalanceNegative` | 3215 | — |
| `CrossSaveAccountNotAuthenticated` | 3216 | — |
| `ErrorOneAccountAlreadyActive` | 3217 | — |
| `ErrorOneAccountDestinyRestriction` | 3218 | — |
| `CrossSaveMustMigrateToSteam` | 3219 | — |
| `CrossSaveSteamAlreadyPaired` | 3220 | — |
| `CrossSaveCannotPairJustSteamAndBlizzard` | 3221 | — |
| `CrossSaveCannotPairSteamAloneBeforeShadowkeep` | 3222 | — |
| `AuthVerificationNotLinkedToAccount` | 3300 | — |
| `PCMigrationMissingBlizzard` | 3400 | — |
| `PCMigrationMissingSteam` | 3401 | — |
| `PCMigrationInvalidBlizzard` | 3402 | — |
| `PCMigrationInvalidSteam` | 3403 | — |
| `PCMigrationUnknownFailure` | 3404 | — |
| `PCMigrationUnknownException` | 3405 | — |
| `PCMigrationNotLinked` | 3406 | — |
| `PCMigrationAccountsAlreadyUsed` | 3407 | — |
| `PCMigrationStepFailed` | 3408 | — |
| `PCMigrationInvalidBlizzardCrossSaveState` | 3409 | — |
| `PCMigrationDestinationBanned` | 3410 | — |
| `PCMigrationDestinyFailure` | 3411 | — |
| `PCMigrationSilverTransferFailed` | 3412 | — |
| `PCMigrationEntitlementTransferFailed` | 3413 | — |
| `PCMigrationCannotStompClanFounder` | 3414 | — |
| `UnsupportedBrowser` | 3500 | — |
| `StadiaAccountRequired` | 3600 | — |
| `ErrorPhoneValidationTooManyUses` | 3702 | — |
| `ErrorPhoneValidationNoAssociatedPhone` | 3703 | — |
| `ErrorPhoneValidationCodeInvalid` | 3705 | — |
| `ErrorPhoneValidationBanned` | 3706 | — |
| `ErrorPhoneValidationCodeTooRecentlySent` | 3707 | — |
| `ErrorPhoneValidationCodeExpired` | 3708 | — |
| `ErrorPhoneValidationInvalidNumberType` | 3709 | — |
| `ErrorPhoneValidationCodeTooRecentlyChecked` | 3710 | — |
| `ErrorPhoneValidationRecentlyPlayedDestiny2AccountRequired` | 3711 | — |
| `ApplePushErrorUnknown` | 3800 | — |
| `ApplePushErrorNull` | 3801 | — |
| `ApplePushErrorTimeout` | 3802 | — |
| `ApplePushBadRequest` | 3803 | — |
| `ApplePushFailedAuth` | 3804 | — |
| `ApplePushThrottled` | 3805 | — |
| `ApplePushServiceUnavailable` | 3806 | — |
| `NotAnImageOrVideo` | 3807 | — |
| `ErrorBungieFriendsBlockFailed` | 3900 | — |
| `ErrorBungieFriendsAutoReject` | 3901 | — |
| `ErrorBungieFriendsNoRequestFound` | 3902 | — |
| `ErrorBungieFriendsAlreadyFriends` | 3903 | — |
| `ErrorBungieFriendsUnableToRemoveRequest` | 3904 | — |
| `ErrorBungieFriendsUnableToRemove` | 3905 | — |
| `ErrorBungieFriendsIdenticalSourceTarget` | 3906 | — |
| `ErrorBungieFriendsSelf` | 3907 | — |
| `ErrorBungieBlockSelf` | 3908 | — |
| `ErrorBungieFriendsListFull` | 3910 | — |
| `ErrorBungieBlockListFull` | 3911 | — |
| `ErrorBungieFriendNotFound` | 3912 | — |
| `ErrorBungieFriendInvalidMembershipType` | 3913 | — |
| `ErrorEgsUnknown` | 4000 | — |
| `ErrorEgsBadRequest` | 4001 | — |
| `ErrorEgsNotAuthorized` | 4002 | — |
| `ErrorEgsForbidden` | 4003 | — |
| `ErrorEgsAccountNotFound` | 4004 | — |
| `ErrorEgsWebException` | 4005 | — |
| `ErrorEgsUnavailable` | 4006 | — |
| `ErrorEgsJwksMissing` | 4007 | — |
| `ErrorEgsJwtMalformedHeader` | 4008 | — |
| `ErrorEgsJwtMalformedPayload` | 4009 | — |

### Namespace: Fireteam

#### Fireteam.FireteamDateRangeEnumeration

**Enum** (`byte`)

| Value | # | Description |
| --- | --- | --- |
| `All` | 0 | — |
| `Now` | 1 | — |
| `TwentyFourHours` | 2 | — |
| `FortyEightHours` | 3 | — |
| `ThisWeek` | 4 | — |

#### Fireteam.FireteamPlatformEnumeration

**Enum** (`byte`)

| Value | # | Description |
| --- | --- | --- |
| `Any` | 0 | — |
| `Playstation4` | 1 | — |
| `XboxOne` | 2 | — |
| `Blizzard` | 3 | — |
| `Steam` | 4 | — |
| `Stadia` | 5 | — |
| `Egs` | 6 | — |

#### Fireteam.FireteamPublicSearchOptionEnumeration

**Enum** (`byte`)

| Value | # | Description |
| --- | --- | --- |
| `PublicAndPrivate` | 0 | — |
| `PublicOnly` | 1 | — |
| `PrivateOnly` | 2 | — |

#### Fireteam.FireteamSlotSearchEnumeration

**Enum** (`byte`)

| Value | # | Description |
| --- | --- | --- |
| `NoSlotRestriction` | 0 | — |
| `HasOpenPlayerSlots` | 1 | — |
| `HasOpenPlayerOrAltSlots` | 2 | — |

#### Fireteam.FireteamSummary

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `fireteamId` | int64 | — |
| `groupId` | int64 | — |
| `platform` | byte | — |
| `activityType` | int32 | — |
| `isImmediate` | boolean | — |
| `scheduledTime` | date-time? | — |
| `ownerMembershipId` | int64 | — |
| `playerSlotCount` | int32 | — |
| `alternateSlotCount` | int32? | — |
| `availablePlayerSlotCount` | int32 | — |
| `availableAlternateSlotCount` | int32 | — |
| `title` | string | — |
| `dateCreated` | date-time | — |
| `dateModified` | date-time? | — |
| `isPublic` | boolean | — |
| `locale` | string | — |
| `isValid` | boolean | — |
| `datePlayerModified` | date-time | — |
| `titleBeforeModeration` | string | — |
| `ownerCurrentGuardianRankSnapshot` | int32 → DestinyGuardianRankDefinition | — |
| `ownerHighestLifetimeGuardianRankSnapshot` | int32 → DestinyGuardianRankDefinition | — |
| `ownerTotalCommendationScoreSnapshot` | int32 | — |

#### Fireteam.FireteamResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `Summary` | Fireteam.FireteamSummary | — |
| `Members` | array&lt;Fireteam.FireteamMember&gt; | — |
| `Alternates` | array&lt;Fireteam.FireteamMember&gt; | — |

#### Fireteam.FireteamMember

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `destinyUserInfo` | Fireteam.FireteamUserInfoCard | — |
| `bungieNetUserInfo` | User.UserInfoCard | — |
| `characterId` | int64 | — |
| `dateJoined` | date-time | — |
| `hasMicrophone` | boolean | — |
| `lastPlatformInviteAttemptDate` | date-time | — |
| `lastPlatformInviteAttemptResult` | byte | — |

#### Fireteam.FireteamUserInfoCard

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `FireteamDisplayName` | string | — |
| `FireteamMembershipType` | int32 | — |
| `supplementalDisplayName` | string | A platform specific additional display name - ex: psn Real Name, bnet Unique Name, etc. |
| `iconPath` | string | URL the Icon if available. |
| `crossSaveOverride` | int32 | If there is a cross save override in effect, this value will tell you the type that is overridding this one. |
| `applicableMembershipTypes` | array&lt;int32&gt; | The list of Membership Types indicating the platforms on which this Membership can be used. Not in Cross Save = its original membership type. Cross Save Primary = Any membership types it is overridding, and its original membership type Cross Save Overridden = Empty list |
| `isPublic` | boolean | If True, this is a public user membership. |
| `membershipType` | int32 | Type of the membership. Not necessarily the native type. |
| `membershipId` | int64 | Membership ID as they user is known in the Accounts service |
| `displayName` | string | Display Name the player has chosen for themselves. The display name is optional when the data type is used as input to a platform API. |
| `bungieGlobalDisplayName` | string | The bungie global display name, if set. |
| `bungieGlobalDisplayNameCode` | int16? | The bungie global display name code, if set. |

#### Fireteam.FireteamPlatformInviteResultEnumeration

**Enum** (`byte`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Success` | 1 | — |
| `AlreadyInFireteam` | 2 | — |
| `Throttled` | 3 | — |
| `ServiceError` | 4 | — |

### Namespace: Forum

#### Forum.ForumTopicsCategoryFiltersEnumEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Links` | 1 | — |
| `Questions` | 2 | — |
| `AnsweredQuestions` | 4 | — |
| `Media` | 8 | — |
| `TextOnly` | 16 | — |
| `Announcement` | 32 | — |
| `BungieOfficial` | 64 | — |
| `Polls` | 128 | — |

#### Forum.ForumTopicsQuickDateEnumEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `All` | 0 | — |
| `LastYear` | 1 | — |
| `LastMonth` | 2 | — |
| `LastWeek` | 3 | — |
| `LastDay` | 4 | — |

#### Forum.ForumTopicsSortEnumEnumeration

**Enum** (`byte`)

| Value | # | Description |
| --- | --- | --- |
| `Default` | 0 | — |
| `LastReplied` | 1 | — |
| `MostReplied` | 2 | — |
| `Popularity` | 3 | — |
| `Controversiality` | 4 | — |
| `Liked` | 5 | — |
| `HighestRated` | 6 | — |
| `MostUpvoted` | 7 | — |

#### Forum.PostResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `lastReplyTimestamp` | date-time | — |
| `IsPinned` | boolean | — |
| `urlMediaType` | int32 | — |
| `thumbnail` | string | — |
| `popularity` | int32 | — |
| `isActive` | boolean | — |
| `isAnnouncement` | boolean | — |
| `userRating` | int32 | — |
| `userHasRated` | boolean | — |
| `userHasMutedPost` | boolean | — |
| `latestReplyPostId` | int64 | — |
| `latestReplyAuthorId` | int64 | — |
| `ignoreStatus` | Ignores.IgnoreResponse | — |
| `locale` | string | — |

#### Forum.ForumMediaTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Image` | 1 | — |
| `Video` | 2 | — |
| `Youtube` | 3 | — |

#### Forum.ForumPostPopularityEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Empty` | 0 | — |
| `Default` | 1 | — |
| `Discussed` | 2 | — |
| `CoolStory` | 3 | — |
| `HeatingUp` | 4 | — |
| `Hot` | 5 | — |

#### Forum.PostSearchResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `relatedPosts` | array&lt;Forum.PostResponse&gt; | — |
| `authors` | array&lt;User.GeneralUser&gt; | — |
| `groups` | array&lt;GroupsV2.GroupResponse&gt; | — |
| `searchedTags` | array&lt;Tags.Models.Contracts.TagResponse&gt; | — |
| `polls` | array&lt;Forum.PollResponse&gt; | — |
| `recruitmentDetails` | array&lt;Forum.ForumRecruitmentDetail&gt; | — |
| `availablePages` | int32? | — |
| `results` | array&lt;Forum.PostResponse&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### Forum.PollResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `topicId` | int64 | — |
| `results` | array&lt;Forum.PollResult&gt; | — |
| `totalVotes` | int32 | — |

#### Forum.PollResult

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `answerText` | string | — |
| `answerSlot` | int32 | — |
| `lastVoteDate` | date-time | — |
| `votes` | int32 | — |
| `requestingUserVoted` | boolean | — |

#### Forum.ForumRecruitmentDetail

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `topicId` | int64 | — |
| `microphoneRequired` | boolean | — |
| `intensity` | byte | — |
| `tone` | byte | — |
| `approved` | boolean | — |
| `conversationId` | int64? | — |
| `playerSlotsTotal` | int32 | — |
| `playerSlotsRemaining` | int32 | — |
| `Fireteam` | array&lt;User.GeneralUser&gt; | — |
| `kickedPlayerIds` | array&lt;int64&gt; | — |

#### Forum.ForumRecruitmentIntensityLabelEnumeration

**Enum** (`byte`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Casual` | 1 | — |
| `Professional` | 2 | — |

#### Forum.ForumRecruitmentToneLabelEnumeration

**Enum** (`byte`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `FamilyFriendly` | 1 | — |
| `Rowdy` | 2 | — |

#### Forum.ForumPostSortEnumEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Default` | 0 | — |
| `OldestFirst` | 1 | — |

#### Forum.CommunityContentSortModeEnumeration

**Enum** (`byte`)

| Value | # | Description |
| --- | --- | --- |
| `Trending` | 0 | — |
| `Latest` | 1 | — |
| `HighestRated` | 2 | — |

### Namespace: Forums

#### Forums.ForumPostCategoryEnumsEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `TextOnly` | 1 | — |
| `Media` | 2 | — |
| `Link` | 4 | — |
| `Poll` | 8 | — |
| `Question` | 16 | — |
| `Answered` | 32 | — |
| `Announcement` | 64 | — |
| `ContentComment` | 128 | — |
| `BungieOfficial` | 256 | — |
| `NinjaOfficial` | 512 | — |
| `Recruitment` | 1024 | — |

#### Forums.ForumFlagsEnumEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `BungieStaffPost` | 1 | — |
| `ForumNinjaPost` | 2 | — |
| `ForumMentorPost` | 4 | — |
| `TopicBungieStaffPosted` | 8 | — |
| `TopicBungieVolunteerPosted` | 16 | — |
| `QuestionAnsweredByBungie` | 32 | — |
| `QuestionAnsweredByNinja` | 64 | — |
| `CommunityContent` | 128 | — |

### Namespace: GroupsV2

#### GroupsV2.GroupUserInfoCard

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `LastSeenDisplayName` | string | This will be the display name the clan server last saw the user as. If the account is an active cross save override, this will be the display name to use. Otherwise, this will match the displayName property. |
| `LastSeenDisplayNameType` | int32 | The platform of the LastSeenDisplayName |
| `supplementalDisplayName` | string | A platform specific additional display name - ex: psn Real Name, bnet Unique Name, etc. |
| `iconPath` | string | URL the Icon if available. |
| `crossSaveOverride` | int32 | If there is a cross save override in effect, this value will tell you the type that is overridding this one. |
| `applicableMembershipTypes` | array&lt;int32&gt; | The list of Membership Types indicating the platforms on which this Membership can be used. Not in Cross Save = its original membership type. Cross Save Primary = Any membership types it is overridding, and its original membership type Cross Save Overridden = Empty list |
| `isPublic` | boolean | If True, this is a public user membership. |
| `membershipType` | int32 | Type of the membership. Not necessarily the native type. |
| `membershipId` | int64 | Membership ID as they user is known in the Accounts service |
| `displayName` | string | Display Name the player has chosen for themselves. The display name is optional when the data type is used as input to a platform API. |
| `bungieGlobalDisplayName` | string | The bungie global display name, if set. |
| `bungieGlobalDisplayNameCode` | int16? | The bungie global display name code, if set. |

#### GroupsV2.GroupResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `detail` | GroupsV2.GroupV2 | — |
| `founder` | GroupsV2.GroupMember | — |
| `alliedIds` | array&lt;int64&gt; | — |
| `parentGroup` | GroupsV2.GroupV2 | — |
| `allianceStatus` | int32 | — |
| `groupJoinInviteCount` | int32 | — |
| `currentUserMembershipsInactiveForDestiny` | boolean | A convenience property that indicates if every membership you (the current user) have that is a part of this group are part of an account that is considered inactive - for example, overridden accounts in Cross Save. |
| `currentUserMemberMap` | Mapping&lt;int32, GroupsV2.GroupMember&gt; | This property will be populated if the authenticated user is a member of the group. Note that because of account linking, a user can sometimes be part of a clan more than once. As such, this returns the highest member type available. |
| `currentUserPotentialMemberMap` | Mapping&lt;int32, GroupsV2.GroupPotentialMember&gt; | This property will be populated if the authenticated user is an applicant or has an outstanding invitation to join. Note that because of account linking, a user can sometimes be part of a clan more than once. |

#### GroupsV2.GroupV2

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `groupId` | int64 | — |
| `name` | string | — |
| `groupType` | int32 | — |
| `membershipIdCreated` | int64 | — |
| `creationDate` | date-time | — |
| `modificationDate` | date-time | — |
| `about` | string | — |
| `tags` | array&lt;string&gt; | — |
| `memberCount` | int32 | — |
| `isPublic` | boolean | — |
| `isPublicTopicAdminOnly` | boolean | — |
| `motto` | string | — |
| `allowChat` | boolean | — |
| `isDefaultPostPublic` | boolean | — |
| `chatSecurity` | int32 | — |
| `locale` | string | — |
| `avatarImageIndex` | int32 | — |
| `homepage` | int32 | — |
| `membershipOption` | int32 | — |
| `defaultPublicity` | int32 | — |
| `theme` | string | — |
| `bannerPath` | string | — |
| `avatarPath` | string | — |
| `conversationId` | int64 | — |
| `enableInvitationMessagingForAdmins` | boolean | — |
| `banExpireDate` | date-time? | — |
| `features` | GroupsV2.GroupFeatures | — |
| `remoteGroupId` | int64? | — |
| `clanInfo` | GroupsV2.GroupV2ClanInfoAndInvestment | — |

#### GroupsV2.GroupTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `General` | 0 | — |
| `Clan` | 1 | — |

#### GroupsV2.ChatSecuritySettingEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Group` | 0 | — |
| `Admins` | 1 | — |

#### GroupsV2.GroupHomepageEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Wall` | 0 | — |
| `Forum` | 1 | — |
| `AllianceForum` | 2 | — |

#### GroupsV2.MembershipOptionEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Reviewed` | 0 | — |
| `Open` | 1 | — |
| `Closed` | 2 | — |

#### GroupsV2.GroupPostPublicityEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Public` | 0 | — |
| `Alliance` | 1 | — |
| `Private` | 2 | — |

#### GroupsV2.GroupFeatures

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `maximumMembers` | int32 | — |
| `maximumMembershipsOfGroupType` | int32 | Maximum number of groups of this type a typical membership may join. For example, a user may join about 50 General groups with their Bungie.net account. They may join one clan per Destiny membership. |
| `capabilities` | int32 | — |
| `membershipTypes` | array&lt;int32&gt; | — |
| `invitePermissionOverride` | boolean | Minimum Member Level allowed to invite new members to group Always Allowed: Founder, Acting Founder True means admins have this power, false means they don't Default is false for clans, true for groups. |
| `updateCulturePermissionOverride` | boolean | Minimum Member Level allowed to update group culture Always Allowed: Founder, Acting Founder True means admins have this power, false means they don't Default is false for clans, true for groups. |
| `hostGuidedGamePermissionOverride` | int32 | Minimum Member Level allowed to host guided games Always Allowed: Founder, Acting Founder, Admin Allowed Overrides: None, Member, Beginner Default is Member for clans, None for groups, although this means nothing for groups. |
| `updateBannerPermissionOverride` | boolean | Minimum Member Level allowed to update banner Always Allowed: Founder, Acting Founder True means admins have this power, false means they don't Default is false for clans, true for groups. |
| `joinLevel` | int32 | Level to join a member at when accepting an invite, application, or joining an open clan Default is Beginner. |

#### GroupsV2.CapabilitiesEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Leaderboards` | 1 | — |
| `Callsign` | 2 | — |
| `OptionalConversations` | 4 | — |
| `ClanBanner` | 8 | — |
| `D2InvestmentData` | 16 | — |
| `Tags` | 32 | — |
| `Alliances` | 64 | — |

#### GroupsV2.HostGuidedGamesPermissionLevelEnumeration

**Enum** (`int32`)

Used for setting the guided game permission level override (admins and founders can always host guided games).

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Beginner` | 1 | — |
| `Member` | 2 | — |

#### GroupsV2.RuntimeGroupMemberTypeEnumeration

**Enum** (`int32`)

The member levels used by all V2 Groups API. Individual group types use their own mappings in their native storage (general uses BnetDbGroupMemberType and D2 clans use ClanMemberLevel), but they are all translated to this in the runtime api. These runtime values should NEVER be stored anywhere, so the values can be changed as necessary.

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Beginner` | 1 | — |
| `Member` | 2 | — |
| `Admin` | 3 | — |
| `ActingFounder` | 4 | — |
| `Founder` | 5 | — |

#### GroupsV2.GroupV2ClanInfo

**Type:** object

This contract contains clan-specific group information. It does not include any investment data.

| Property | Type | Description |
| --- | --- | --- |
| `clanCallsign` | string | — |
| `clanBannerData` | GroupsV2.ClanBanner | — |

#### GroupsV2.ClanBanner

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `decalId` | uint32 | — |
| `decalColorId` | uint32 | — |
| `decalBackgroundColorId` | uint32 | — |
| `gonfalonId` | uint32 | — |
| `gonfalonColorId` | uint32 | — |
| `gonfalonDetailId` | uint32 | — |
| `gonfalonDetailColorId` | uint32 | — |

#### GroupsV2.GroupV2ClanInfoAndInvestment

**Type:** object

The same as GroupV2ClanInfo, but includes any investment data.

| Property | Type | Description |
| --- | --- | --- |
| `d2ClanProgressions` | Mapping&lt;uint32, Destiny.DestinyProgression&gt; | — |
| `clanCallsign` | string | — |
| `clanBannerData` | GroupsV2.ClanBanner | — |

#### GroupsV2.GroupUserBase

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `groupId` | int64 | — |
| `destinyUserInfo` | GroupsV2.GroupUserInfoCard | — |
| `bungieNetUserInfo` | User.UserInfoCard | — |
| `joinDate` | date-time | — |

#### GroupsV2.GroupMember

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `memberType` | int32 | — |
| `isOnline` | boolean | — |
| `lastOnlineStatusChange` | int64 | — |
| `groupId` | int64 | — |
| `destinyUserInfo` | GroupsV2.GroupUserInfoCard | — |
| `bungieNetUserInfo` | User.UserInfoCard | — |
| `joinDate` | date-time | — |

#### GroupsV2.GroupAllianceStatusEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Unallied` | 0 | — |
| `Parent` | 1 | — |
| `Child` | 2 | — |

#### GroupsV2.GroupPotentialMember

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `potentialStatus` | int32 | — |
| `groupId` | int64 | — |
| `destinyUserInfo` | GroupsV2.GroupUserInfoCard | — |
| `bungieNetUserInfo` | User.UserInfoCard | — |
| `joinDate` | date-time | — |

#### GroupsV2.GroupPotentialMemberStatusEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Applicant` | 1 | — |
| `Invitee` | 2 | — |

#### GroupsV2.GroupDateRangeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `All` | 0 | — |
| `PastDay` | 1 | — |
| `PastWeek` | 2 | — |
| `PastMonth` | 3 | — |
| `PastYear` | 4 | — |

#### GroupsV2.GroupV2Card

**Type:** object

A small infocard of group information, usually used for when a list of groups are returned

| Property | Type | Description |
| --- | --- | --- |
| `groupId` | int64 | — |
| `name` | string | — |
| `groupType` | int32 | — |
| `creationDate` | date-time | — |
| `about` | string | — |
| `motto` | string | — |
| `memberCount` | int32 | — |
| `locale` | string | — |
| `membershipOption` | int32 | — |
| `capabilities` | int32 | — |
| `remoteGroupId` | int64? | — |
| `clanInfo` | GroupsV2.GroupV2ClanInfo | — |
| `avatarPath` | string | — |
| `theme` | string | — |

#### GroupsV2.GroupSearchResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;GroupsV2.GroupV2Card&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### GroupsV2.GroupQuery

**Type:** object

NOTE: GroupQuery, as of Destiny 2, has essentially two totally different and incompatible "modes". If you are querying for a group, you can pass any of the properties below. If you are querying for a Clan, you MUST NOT pass any of the following properties (they must be null or undefined in your request, not just empty string/default values): - groupMemberCountFilter - localeFilter - tagText If you pass these, you will get a useless InvalidParameters error.

| Property | Type | Description |
| --- | --- | --- |
| `name` | string | — |
| `groupType` | int32 | — |
| `creationDate` | int32 | — |
| `sortBy` | int32 | — |
| `groupMemberCountFilter` | int32? | — |
| `localeFilter` | string | — |
| `tagText` | string | — |
| `itemsPerPage` | int32 | — |
| `currentPage` | int32 | — |
| `requestContinuationToken` | string | — |

#### GroupsV2.GroupSortByEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Name` | 0 | — |
| `Date` | 1 | — |
| `Popularity` | 2 | — |
| `Id` | 3 | — |

#### GroupsV2.GroupMemberCountFilterEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `All` | 0 | — |
| `OneToTen` | 1 | — |
| `ElevenToOneHundred` | 2 | — |
| `GreaterThanOneHundred` | 3 | — |

#### GroupsV2.GroupNameSearchRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `groupName` | string | — |
| `groupType` | int32 | — |

#### GroupsV2.GroupOptionalConversation

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `groupId` | int64 | — |
| `conversationId` | int64 | — |
| `chatEnabled` | boolean | — |
| `chatName` | string | — |
| `chatSecurity` | int32 | — |

#### GroupsV2.GroupEditAction

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `name` | string | — |
| `about` | string | — |
| `motto` | string | — |
| `theme` | string | — |
| `avatarImageIndex` | int32? | — |
| `tags` | string | — |
| `isPublic` | boolean? | — |
| `membershipOption` | int32? | — |
| `isPublicTopicAdminOnly` | boolean? | — |
| `allowChat` | boolean? | — |
| `chatSecurity` | int32? | — |
| `callsign` | string | — |
| `locale` | string | — |
| `homepage` | int32? | — |
| `enableInvitationMessagingForAdmins` | boolean? | — |
| `defaultPublicity` | int32? | — |

#### GroupsV2.GroupOptionsEditAction

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `InvitePermissionOverride` | boolean? | Minimum Member Level allowed to invite new members to group Always Allowed: Founder, Acting Founder True means admins have this power, false means they don't Default is false for clans, true for groups. |
| `UpdateCulturePermissionOverride` | boolean? | Minimum Member Level allowed to update group culture Always Allowed: Founder, Acting Founder True means admins have this power, false means they don't Default is false for clans, true for groups. |
| `HostGuidedGamePermissionOverride` | int32? | Minimum Member Level allowed to host guided games Always Allowed: Founder, Acting Founder, Admin Allowed Overrides: None, Member, Beginner Default is Member for clans, None for groups, although this means nothing for groups. |
| `UpdateBannerPermissionOverride` | boolean? | Minimum Member Level allowed to update banner Always Allowed: Founder, Acting Founder True means admins have this power, false means they don't Default is false for clans, true for groups. |
| `JoinLevel` | int32? | Level to join a member at when accepting an invite, application, or joining an open clan Default is Beginner. |

#### GroupsV2.GroupOptionalConversationAddRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `chatName` | string | — |
| `chatSecurity` | int32 | — |

#### GroupsV2.GroupOptionalConversationEditRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `chatEnabled` | boolean? | — |
| `chatName` | string | — |
| `chatSecurity` | int32? | — |

#### GroupsV2.GroupMemberLeaveResult

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `group` | GroupsV2.GroupV2 | — |
| `groupDeleted` | boolean | — |

#### GroupsV2.GroupBanRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `comment` | string | — |
| `length` | int32 | — |

#### GroupsV2.GroupBan

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `groupId` | int64 | — |
| `lastModifiedBy` | User.UserInfoCard | — |
| `createdBy` | User.UserInfoCard | — |
| `dateBanned` | date-time | — |
| `dateExpires` | date-time | — |
| `comment` | string | — |
| `bungieNetUserInfo` | User.UserInfoCard | — |
| `destinyUserInfo` | GroupsV2.GroupUserInfoCard | — |

#### GroupsV2.GroupEditHistory

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `groupId` | int64 | — |
| `name` | string | — |
| `nameEditors` | int64? | — |
| `about` | string | — |
| `aboutEditors` | int64? | — |
| `motto` | string | — |
| `mottoEditors` | int64? | — |
| `clanCallsign` | string | — |
| `clanCallsignEditors` | int64? | — |
| `editDate` | date-time? | — |
| `groupEditors` | array&lt;User.UserInfoCard&gt; | — |

#### GroupsV2.GroupMemberApplication

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `groupId` | int64 | — |
| `creationDate` | date-time | — |
| `resolveState` | int32 | — |
| `resolveDate` | date-time? | — |
| `resolvedByMembershipId` | int64? | — |
| `requestMessage` | string | — |
| `resolveMessage` | string | — |
| `destinyUserInfo` | GroupsV2.GroupUserInfoCard | — |
| `bungieNetUserInfo` | User.UserInfoCard | — |

#### GroupsV2.GroupApplicationResolveStateEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Unresolved` | 0 | — |
| `Accepted` | 1 | — |
| `Denied` | 2 | — |
| `Rescinded` | 3 | — |

#### GroupsV2.GroupApplicationRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `message` | string | — |

#### GroupsV2.GroupApplicationListRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `memberships` | array&lt;User.UserMembership&gt; | — |
| `message` | string | — |

#### GroupsV2.GroupsForMemberFilterEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `All` | 0 | — |
| `Founded` | 1 | — |
| `NonFounded` | 2 | — |

#### GroupsV2.GroupMembershipBase

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `group` | GroupsV2.GroupV2 | — |

#### GroupsV2.GroupMembership

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `member` | GroupsV2.GroupMember | — |
| `group` | GroupsV2.GroupV2 | — |

#### GroupsV2.GroupMembershipSearchResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;GroupsV2.GroupMembership&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### GroupsV2.GetGroupsForMemberResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `areAllMembershipsInactive` | Mapping&lt;int64, boolean&gt; | A convenience property that indicates if every membership this user has that is a part of this group are part of an account that is considered inactive - for example, overridden accounts in Cross Save. The key is the Group ID for the group being checked, and the value is true if the users' memberships for that group are all inactive. |
| `results` | array&lt;GroupsV2.GroupMembership&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### GroupsV2.GroupPotentialMembership

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `member` | GroupsV2.GroupPotentialMember | — |
| `group` | GroupsV2.GroupV2 | — |

#### GroupsV2.GroupPotentialMembershipSearchResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `results` | array&lt;GroupsV2.GroupPotentialMembership&gt; | — |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### GroupsV2.GroupApplicationResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `resolution` | int32 | — |

### Namespace: Ignores

#### Ignores.IgnoreResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `isIgnored` | boolean | — |
| `ignoreFlags` | int32 | — |

#### Ignores.IgnoreStatusEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `NotIgnored` | 0 | — |
| `IgnoredUser` | 1 | — |
| `IgnoredGroup` | 2 | — |
| `IgnoredByGroup` | 4 | — |
| `IgnoredPost` | 8 | — |
| `IgnoredTag` | 16 | — |
| `IgnoredGlobal` | 32 | — |

#### Ignores.IgnoreLengthEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Week` | 1 | — |
| `TwoWeeks` | 2 | — |
| `ThreeWeeks` | 3 | — |
| `Month` | 4 | — |
| `ThreeMonths` | 5 | — |
| `SixMonths` | 6 | — |
| `Year` | 7 | — |
| `Forever` | 8 | — |
| `ThreeMinutes` | 9 | — |
| `Hour` | 10 | — |
| `ThirtyDays` | 11 | — |

### Namespace: Interpolation

#### Interpolation.InterpolationPoint

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `value` | int32 | — |
| `weight` | int32 | — |

#### Interpolation.InterpolationPointFloat

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `value` | float | — |
| `weight` | float | — |

### Namespace: Links

#### Links.HyperlinkReference

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `title` | string | — |
| `url` | string | — |

### Namespace: Queries

#### Queries.SearchResult

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `totalResults` | int32 | — |
| `hasMore` | boolean | — |
| `query` | Queries.PagedQuery | — |
| `replacementContinuationToken` | string | — |
| `useTotalResults` | boolean | If useTotalResults is true, then totalResults represents an accurate count. If False, it does not, and may be estimated/only the size of the current page. Either way, you should probably always only trust hasMore. This is a long-held historical throwback to when we used to do paging with known total results. Those queries toasted our database, and we were left to hastily alter our endpoints and create backward- compatible shims, of which useTotalResults is one. |

#### Queries.PagedQuery

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `itemsPerPage` | int32 | — |
| `currentPage` | int32 | — |
| `requestContinuationToken` | string | — |

### Namespace: Social

#### Social.Friends.BungieFriendListResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `friends` | array&lt;Social.Friends.BungieFriend&gt; | — |

#### Social.Friends.BungieFriend

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `lastSeenAsMembershipId` | int64 | — |
| `lastSeenAsBungieMembershipType` | int32 | — |
| `bungieGlobalDisplayName` | string | — |
| `bungieGlobalDisplayNameCode` | int16? | — |
| `onlineStatus` | int32 | — |
| `onlineTitle` | int32 | — |
| `relationship` | int32 | — |
| `bungieNetUser` | User.GeneralUser | — |

#### Social.Friends.PresenceStatusEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `OfflineOrUnknown` | 0 | — |
| `Online` | 1 | — |

#### Social.Friends.PresenceOnlineStateFlagsEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Destiny1` | 1 | — |
| `Destiny2` | 2 | — |

#### Social.Friends.FriendRelationshipStateEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Unknown` | 0 | — |
| `Friend` | 1 | — |
| `IncomingRequest` | 2 | — |
| `OutgoingRequest` | 3 | — |

#### Social.Friends.BungieFriendRequestListResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `incomingRequests` | array&lt;Social.Friends.BungieFriend&gt; | — |
| `outgoingRequests` | array&lt;Social.Friends.BungieFriend&gt; | — |

#### Social.Friends.PlatformFriendTypeEnumeration

**Enum** (`int32`)

| Value | # | Description |
| --- | --- | --- |
| `Unknown` | 0 | — |
| `Xbox` | 1 | — |
| `PSN` | 2 | — |
| `Steam` | 3 | — |
| `Egs` | 4 | — |

#### Social.Friends.PlatformFriendResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `itemsPerPage` | int32 | — |
| `currentPage` | int32 | — |
| `hasMore` | boolean | — |
| `platformFriends` | array&lt;Social.Friends.PlatformFriend&gt; | — |

#### Social.Friends.PlatformFriend

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `platformDisplayName` | string | — |
| `friendPlatform` | int32 | — |
| `destinyMembershipId` | int64? | — |
| `destinyMembershipType` | int32? | — |
| `bungieNetMembershipId` | int64? | — |
| `bungieGlobalDisplayName` | string | — |
| `bungieGlobalDisplayNameCode` | int16? | — |

### Namespace: Streaming

#### Streaming.DropStateEnumEnumeration

**Enum** (`byte`)

| Value | # | Description |
| --- | --- | --- |
| `Claimed` | 0 | — |
| `Applied` | 1 | — |
| `Fulfilled` | 2 | — |

### Namespace: Tags

#### Tags.Models.Contracts.TagResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `tagText` | string | — |
| `ignoreStatus` | Ignores.IgnoreResponse | — |

### Namespace: Tokens

#### Tokens.PartnerOfferClaimRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `PartnerOfferId` | string | — |
| `BungieNetMembershipId` | int64 | — |
| `TransactionId` | string | — |

#### Tokens.PartnerOfferSkuHistoryResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `SkuIdentifier` | string | — |
| `LocalizedName` | string | — |
| `LocalizedDescription` | string | — |
| `ClaimDate` | date-time | — |
| `AllOffersApplied` | boolean | — |
| `TransactionId` | string | — |
| `SkuOffers` | array&lt;Tokens.PartnerOfferHistoryResponse&gt; | — |

#### Tokens.PartnerOfferHistoryResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `PartnerOfferKey` | string | — |
| `MembershipId` | int64? | — |
| `MembershipType` | int32? | — |
| `LocalizedName` | string | — |
| `LocalizedDescription` | string | — |
| `IsConsumable` | boolean | — |
| `QuantityApplied` | int32 | — |
| `ApplyDate` | date-time? | — |

#### Tokens.PartnerRewardHistoryResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `PartnerOffers` | array&lt;Tokens.PartnerOfferSkuHistoryResponse&gt; | — |
| `TwitchDrops` | array&lt;Tokens.TwitchDropHistoryResponse&gt; | — |

#### Tokens.TwitchDropHistoryResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `Title` | string | — |
| `Description` | string | — |
| `CreatedAt` | date-time? | — |
| `ClaimState` | byte? | — |

#### Tokens.BungieRewardDisplay

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `UserRewardAvailabilityModel` | Tokens.UserRewardAvailabilityModel | — |
| `ObjectiveDisplayProperties` | Tokens.RewardDisplayProperties | — |
| `RewardDisplayProperties` | Tokens.RewardDisplayProperties | — |

#### Tokens.UserRewardAvailabilityModel

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `AvailabilityModel` | Tokens.RewardAvailabilityModel | — |
| `IsAvailableForUser` | boolean | — |
| `IsUnlockedForUser` | boolean | — |

#### Tokens.RewardAvailabilityModel

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `HasExistingCode` | boolean | — |
| `RecordDefinitions` | array&lt;Destiny.Definitions.Records.DestinyRecordDefinition&gt; | — |
| `CollectibleDefinitions` | array&lt;Tokens.CollectibleDefinitions&gt; | — |
| `IsOffer` | boolean | — |
| `HasOffer` | boolean | — |
| `OfferApplied` | boolean | — |
| `DecryptedToken` | string | — |
| `IsLoyaltyReward` | boolean | — |
| `ShopifyEndDate` | date-time? | — |
| `GameEarnByDate` | date-time | — |
| `RedemptionEndDate` | date-time | — |

#### Tokens.CollectibleDefinitions

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `CollectibleDefinition` | Destiny.Definitions.Collectibles.DestinyCollectibleDefinition | — |
| `DestinyInventoryItemDefinition` | Destiny.Definitions.DestinyInventoryItemDefinition | — |

#### Tokens.RewardDisplayProperties

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `Name` | string | — |
| `Description` | string | — |
| `ImagePath` | string | — |

### Namespace: Trending

#### Trending.TrendingCategories

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `categories` | array&lt;Trending.TrendingCategory&gt; | — |

#### Trending.TrendingCategory

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `categoryName` | string | — |
| `entries` | SearchResultOfTrendingEntry | — |
| `categoryId` | string | — |

#### Trending.TrendingEntry

**Type:** object

The list entry view for trending items. Returns just enough to show the item on the trending page.

| Property | Type | Description |
| --- | --- | --- |
| `weight` | double | The weighted score of this trending item. |
| `isFeatured` | boolean | — |
| `identifier` | string | We don't know whether the identifier will be a string, a uint, or a long... so we're going to cast it all to a string. But either way, we need any trending item created to have a single unique identifier for its type. |
| `entityType` | int32 | An enum - unfortunately - dictating all of the possible kinds of trending items that you might get in your result set, in case you want to do custom rendering or call to get the details of the item. |
| `displayName` | string | The localized "display name/article title/'primary localized identifier'" of the entity. |
| `tagline` | string | If the entity has a localized tagline/subtitle/motto/whatever, that is found here. |
| `image` | string | — |
| `startDate` | date-time? | — |
| `endDate` | date-time? | — |
| `link` | string | — |
| `webmVideo` | string | If this is populated, the entry has a related WebM video to show. I am 100% certain I am going to regret putting this directly on TrendingEntry, but it will work so yolo |
| `mp4Video` | string | If this is populated, the entry has a related MP4 video to show. I am 100% certain I am going to regret putting this directly on TrendingEntry, but it will work so yolo |
| `featureImage` | string | If isFeatured, this image will be populated with whatever the featured image is. Note that this will likely be a very large image, so don't use it all the time. |
| `items` | array&lt;Trending.TrendingEntry&gt; | If the item is of entityType TrendingEntryType.Container, it may have items - also Trending Entries - contained within it. This is the ordered list of those to display under the Container's header. |
| `creationDate` | date-time? | If the entry has a date at which it was created, this is that date. |

#### Trending.TrendingEntryTypeEnumeration

**Enum** (`int32`)

The known entity types that you can have returned from Trending.

| Value | # | Description |
| --- | --- | --- |
| `News` | 0 | — |
| `DestinyItem` | 1 | — |
| `DestinyActivity` | 2 | — |
| `DestinyRitual` | 3 | — |
| `SupportArticle` | 4 | — |
| `Creation` | 5 | — |
| `Stream` | 6 | — |
| `Update` | 7 | — |
| `Link` | 8 | — |
| `ForumTag` | 9 | — |
| `Container` | 10 | — |
| `Release` | 11 | — |

#### Trending.TrendingDetail

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `identifier` | string | — |
| `entityType` | int32 | — |
| `news` | Trending.TrendingEntryNews | — |
| `support` | Trending.TrendingEntrySupportArticle | — |
| `destinyItem` | Trending.TrendingEntryDestinyItem | — |
| `destinyActivity` | Trending.TrendingEntryDestinyActivity | — |
| `destinyRitual` | Trending.TrendingEntryDestinyRitual | — |
| `creation` | Trending.TrendingEntryCommunityCreation | — |

#### Trending.TrendingEntryNews

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `article` | Content.ContentItemPublicContract | — |

#### Trending.TrendingEntrySupportArticle

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `article` | Content.ContentItemPublicContract | — |

#### Trending.TrendingEntryDestinyItem

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `itemHash` | uint32 | — |

#### Trending.TrendingEntryDestinyActivity

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `activityHash` | uint32 | — |
| `status` | Destiny.Activities.DestinyPublicActivityStatus | — |

#### Trending.TrendingEntryDestinyRitual

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `image` | string | — |
| `icon` | string | — |
| `title` | string | — |
| `subtitle` | string | — |
| `dateStart` | date-time? | — |
| `dateEnd` | date-time? | — |
| `milestoneDetails` | Destiny.Milestones.DestinyPublicMilestone | A destiny event does not necessarily have a related Milestone, but if it does the details will be returned here. |
| `eventContent` | Destiny.Milestones.DestinyMilestoneContent | A destiny event will not necessarily have milestone "custom content", but if it does the details will be here. |

#### Trending.TrendingEntryCommunityCreation

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `media` | string | — |
| `title` | string | — |
| `author` | string | — |
| `authorMembershipId` | int64 | — |
| `postId` | int64 | — |
| `body` | string | — |
| `upvotes` | int32 | — |

### Namespace: User

#### User.UserMembership

**Type:** object

Very basic info about a user as returned by the Account server.

| Property | Type | Description |
| --- | --- | --- |
| `membershipType` | int32 | Type of the membership. Not necessarily the native type. |
| `membershipId` | int64 | Membership ID as they user is known in the Accounts service |
| `displayName` | string | Display Name the player has chosen for themselves. The display name is optional when the data type is used as input to a platform API. |
| `bungieGlobalDisplayName` | string | The bungie global display name, if set. |
| `bungieGlobalDisplayNameCode` | int16? | The bungie global display name code, if set. |

#### User.CrossSaveUserMembership

**Type:** object

Very basic info about a user as returned by the Account server, but including CrossSave information. Do NOT use as a request contract.

| Property | Type | Description |
| --- | --- | --- |
| `crossSaveOverride` | int32 | If there is a cross save override in effect, this value will tell you the type that is overridding this one. |
| `applicableMembershipTypes` | array&lt;int32&gt; | The list of Membership Types indicating the platforms on which this Membership can be used. Not in Cross Save = its original membership type. Cross Save Primary = Any membership types it is overridding, and its original membership type Cross Save Overridden = Empty list |
| `isPublic` | boolean | If True, this is a public user membership. |
| `membershipType` | int32 | Type of the membership. Not necessarily the native type. |
| `membershipId` | int64 | Membership ID as they user is known in the Accounts service |
| `displayName` | string | Display Name the player has chosen for themselves. The display name is optional when the data type is used as input to a platform API. |
| `bungieGlobalDisplayName` | string | The bungie global display name, if set. |
| `bungieGlobalDisplayNameCode` | int16? | The bungie global display name code, if set. |

#### User.UserInfoCard

**Type:** object

This contract supplies basic information commonly used to display a minimal amount of information about a user. Take care to not add more properties here unless the property applies in all (or at least the majority) of the situations where UserInfoCard is used. Avoid adding game specific or platform specific details here. In cases where UserInfoCard is a subset of the data needed in a contract, use UserInfoCard as a property of other contracts.

| Property | Type | Description |
| --- | --- | --- |
| `supplementalDisplayName` | string | A platform specific additional display name - ex: psn Real Name, bnet Unique Name, etc. |
| `iconPath` | string | URL the Icon if available. |
| `crossSaveOverride` | int32 | If there is a cross save override in effect, this value will tell you the type that is overridding this one. |
| `applicableMembershipTypes` | array&lt;int32&gt; | The list of Membership Types indicating the platforms on which this Membership can be used. Not in Cross Save = its original membership type. Cross Save Primary = Any membership types it is overridding, and its original membership type Cross Save Overridden = Empty list |
| `isPublic` | boolean | If True, this is a public user membership. |
| `membershipType` | int32 | Type of the membership. Not necessarily the native type. |
| `membershipId` | int64 | Membership ID as they user is known in the Accounts service |
| `displayName` | string | Display Name the player has chosen for themselves. The display name is optional when the data type is used as input to a platform API. |
| `bungieGlobalDisplayName` | string | The bungie global display name, if set. |
| `bungieGlobalDisplayNameCode` | int16? | The bungie global display name code, if set. |

#### User.GeneralUser

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `membershipId` | int64 | — |
| `uniqueName` | string | — |
| `normalizedName` | string | — |
| `displayName` | string | — |
| `profilePicture` | int32 | — |
| `profileTheme` | int32 | — |
| `userTitle` | int32 | — |
| `successMessageFlags` | int64 | — |
| `isDeleted` | boolean | — |
| `about` | string | — |
| `firstAccess` | date-time? | — |
| `lastUpdate` | date-time? | — |
| `legacyPortalUID` | int64? | — |
| `context` | User.UserToUserContext | — |
| `psnDisplayName` | string | — |
| `xboxDisplayName` | string | — |
| `fbDisplayName` | string | — |
| `showActivity` | boolean? | — |
| `locale` | string | — |
| `localeInheritDefault` | boolean | — |
| `lastBanReportId` | int64? | — |
| `showGroupMessaging` | boolean | — |
| `profilePicturePath` | string | — |
| `profilePictureWidePath` | string | — |
| `profileThemeName` | string | — |
| `userTitleDisplay` | string | — |
| `statusText` | string | — |
| `statusDate` | date-time | — |
| `profileBanExpire` | date-time? | — |
| `blizzardDisplayName` | string | — |
| `steamDisplayName` | string | — |
| `stadiaDisplayName` | string | — |
| `twitchDisplayName` | string | — |
| `cachedBungieGlobalDisplayName` | string | — |
| `cachedBungieGlobalDisplayNameCode` | int16? | — |
| `egsDisplayName` | string | — |

#### User.UserToUserContext

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `isFollowing` | boolean | — |
| `ignoreStatus` | Ignores.IgnoreResponse | — |
| `globalIgnoreEndDate` | date-time? | — |

#### User.Models.GetCredentialTypesForAccountResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `credentialType` | byte | — |
| `credentialDisplayName` | string | — |
| `isPublic` | boolean | — |
| `credentialAsString` | string | — |

#### User.UserMembershipData

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `destinyMemberships` | array&lt;GroupsV2.GroupUserInfoCard&gt; | this allows you to see destiny memberships that are visible and linked to this account (regardless of whether or not they have characters on the world server) |
| `primaryMembershipId` | int64? | If this property is populated, it will have the membership ID of the account considered to be "primary" in this user's cross save relationship. If null, this user has no cross save relationship, nor primary account. |
| `marathonMembershipId` | int64? | If this property is populated, it will have the membershipId for the Marathon Membership on this user's account If null, this user has no Marathon (i.e. "GoliathGame") membership. |
| `bungieNetUser` | User.GeneralUser | — |

#### User.HardLinkedUserMembership

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `membershipType` | int32 | — |
| `membershipId` | int64 | — |
| `CrossSaveOverriddenType` | int32 | — |
| `CrossSaveOverriddenMembershipId` | int64? | — |

#### User.UserSearchResponse

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `searchResults` | array&lt;User.UserSearchResponseDetail&gt; | — |
| `page` | int32 | — |
| `hasMore` | boolean | — |

#### User.UserSearchResponseDetail

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `bungieGlobalDisplayName` | string | — |
| `bungieGlobalDisplayNameCode` | int16? | — |
| `bungieNetMembershipId` | int64? | — |
| `destinyMemberships` | array&lt;User.UserInfoCard&gt; | — |

#### User.UserSearchPrefixRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `displayNamePrefix` | string | — |

#### User.ExactSearchRequest

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `displayName` | string | — |
| `displayNameCode` | int16 | — |

#### User.EmailSettings

**Type:** object

The set of all email subscription/opt-in settings and definitions.

| Property | Type | Description |
| --- | --- | --- |
| `optInDefinitions` | Mapping&lt;string, User.EmailOptInDefinition&gt; | Keyed by the name identifier of the opt-in definition. |
| `subscriptionDefinitions` | Mapping&lt;string, User.EmailSubscriptionDefinition&gt; | Keyed by the name identifier of the Subscription definition. |
| `views` | Mapping&lt;string, User.EmailViewDefinition&gt; | Keyed by the name identifier of the View definition. |

#### User.EmailOptInDefinition

**Type:** object

Defines a single opt-in category: a wide-scoped permission to send emails for the subject related to the opt-in.

| Property | Type | Description |
| --- | --- | --- |
| `name` | string | The unique identifier for this opt-in category. |
| `value` | int64 | The flag value for this opt-in category. For historical reasons, this is defined as a flags enum. |
| `setByDefault` | boolean | If true, this opt-in setting should be set by default in situations where accounts are created without explicit choices about what they're opting into. |
| `dependentSubscriptions` | array&lt;User.EmailSubscriptionDefinition&gt; | Information about the dependent subscriptions for this opt-in. |

#### User.OptInFlagsEnumeration

**Enum** (`int64`)

| Value | # | Description |
| --- | --- | --- |
| `None` | 0 | — |
| `Newsletter` | 1 | — |
| `System` | 2 | — |
| `Marketing` | 4 | — |
| `UserResearch` | 8 | — |
| `CustomerService` | 16 | — |
| `Social` | 32 | — |
| `PlayTests` | 64 | — |
| `PlayTestsLocal` | 128 | — |
| `Careers` | 256 | — |

#### User.EmailSubscriptionDefinition

**Type:** object

Defines a single subscription: permission to send emails for a specific, focused subject (generally timeboxed, such as for a specific release of a product or feature).

| Property | Type | Description |
| --- | --- | --- |
| `name` | string | The unique identifier for this subscription. |
| `localization` | Mapping&lt;string, User.EMailSettingSubscriptionLocalization&gt; | A dictionary of localized text for the EMail Opt-in setting, keyed by the locale. |
| `value` | int64 | The bitflag value for this subscription. Should be a unique power of two value. |

#### User.EMailSettingLocalization

**Type:** object

Localized text relevant to a given EMail setting in a given localization.

| Property | Type | Description |
| --- | --- | --- |
| `title` | string | — |
| `description` | string | — |

#### User.EMailSettingSubscriptionLocalization

**Type:** object

Localized text relevant to a given EMail setting in a given localization. Extra settings specifically for subscriptions.

| Property | Type | Description |
| --- | --- | --- |
| `unknownUserDescription` | string | — |
| `registeredUserDescription` | string | — |
| `unregisteredUserDescription` | string | — |
| `unknownUserActionText` | string | — |
| `knownUserActionText` | string | — |
| `title` | string | — |
| `description` | string | — |

#### User.EmailViewDefinition

**Type:** object

Represents a data-driven view for Email settings. Web/Mobile UI can use this data to show new EMail settings consistently without further manual work.

| Property | Type | Description |
| --- | --- | --- |
| `name` | string | The identifier for this view. |
| `viewSettings` | array&lt;User.EmailViewDefinitionSetting&gt; | The ordered list of settings to show in this view. |

#### User.EmailViewDefinitionSetting

**Type:** object

| Property | Type | Description |
| --- | --- | --- |
| `name` | string | The identifier for this UI Setting, which can be used to relate it to custom strings or other data as desired. |
| `localization` | Mapping&lt;string, User.EMailSettingLocalization&gt; | A dictionary of localized text for the EMail setting, keyed by the locale. |
| `setByDefault` | boolean | If true, this setting should be set by default if the user hasn't chosen whether it's set or cleared yet. |
| `optInAggregateValue` | int64 | The OptInFlags value to set or clear if this setting is set or cleared in the UI. It is the aggregate of all underlying opt-in flags related to this setting. |
| `subscriptions` | array&lt;User.EmailSubscriptionDefinition&gt; | The subscriptions to show as children of this setting, if any. |
