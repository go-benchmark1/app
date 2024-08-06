// Users
const getUsersMe = "/api/users/me"
const postChangePassword = `/api/users/password`
// Templates
const getTemplate = (id) => `/api/templates/${id}`
const getTemplates = "/api/templates"
const postTemplates = "/api/templates"
const putTemplates = (id) => `/api/templates/${id}`
const deleleteTemplates = (id) => `/api/templates/${id}`

// Campaigns
const getCampaign = (id) => `/api/campaigns/${id}`
const getCampaignBounces = (id) => `/api/campaigns/${id}/bounces`
const getCampaignClicks = (id) => `/api/campaigns/${id}/clicks`
const getCampaignComplaints = (id) => `/api/campaigns/${id}/complaints`
const getCampaignOpens = (id) => `/api/campaigns/${id}/opens`
const getCampaigns = "/api/campaigns"
const postCampaigns = "/api/campaigns"
const putCampaigns = (id) => `/api/campaigns/${id}`
const deleteCampaigns = (id) => `/api/campaigns/${id}`
const getCampaignsStats = (id) => `/api/campaigns/${id}/stats`
const postCampaignsStart = (id) => `/api/campaigns/${id}/start`

// Groups
const getGroup = (id) => `/api/segments/${id}`
const getGroupSubscribers = (id) => `/api/segments/${id}/subscribers`
const getGroups = "/api/segments"
const postGroups = "/api/segments"
const putGroups = (id) => `/api/segments/${id}`
const deleteGroups = (id) => `/api/segments/${id}`
const deleteSubscriberFromGroup = (groupId, subscriberId) =>
    `/api/segments/${groupId}/subscribers/${subscriberId}`

// Subscribers
const getSubscriber = (id) => `/api/subscribers/${id}`
const getSubscribers = "/api/subscribers"
const postSubscribers = "/api/subscribers"
const putSubscribers = (id) => `/api/subscribers/${id}`
const getSubscribersExport = `/api/subscribers/export`
const postSubscribersExport = `/api/subscribers/export`
const getSubscribersExportDownload = `/api/subscribers/export/download`
const deleteSubscribers = (id) => `/api/subscribers/${id}`
const deleteSubscribersBulk = `/api/subscribers/bulk-remove`
const postImportSubscribers = `/api/subscribers/import`

// Ses
const getSesKeys = `/api/ses/keys`
const postSesKeys = `/api/ses/keys`
const getSesQuota = `/api/ses/quota`
const deleteSesKeys = `/api/ses/keys`

// Auth
const signup = "/api/signup"
const signInWithGoogle = "api/auth/google"
const signInWithGithub = "api/auth/github"
const signInWithFacebook = "api/auth/facebook"
const logout = "/api/logout"
const forgotPassword = "/api/forgot-password"
const authenticate = "/api/authenticate"
const verifyEmail = "/api/verify-email"
const signInS3 = "/api/s3/sign"

export const endpoints = {
    verifyEmail,
    getUsersMe,
    getTemplate,
    postChangePassword,
    getTemplates,
    postTemplates,
    putTemplates,
    deleleteTemplates,
    getCampaign,
    getCampaignBounces,
    getCampaignClicks,
    getCampaignComplaints,
    getCampaignOpens,
    getCampaigns,
    postCampaigns,
    putCampaigns,
    deleteCampaigns,
    getCampaignsStats,
    postCampaignsStart,
    getGroup,
    getGroupSubscribers,
    getGroups,
    postGroups,
    putGroups,
    deleteGroups,
    getSubscriber,
    getSubscribers,
    postSubscribers,
    putSubscribers,
    deleteSubscribers,
    deleteSubscriberFromGroup,
    deleteSubscribersBulk,
    postImportSubscribers,
    getSubscribersExport,
    getSubscribersExportDownload,
    postSubscribersExport,
    getSesKeys,
    postSesKeys,
    deleteSesKeys,
    getSesQuota,
    signup,
    signInWithGoogle,
    signInWithGithub,
    signInWithFacebook,
    logout,
    forgotPassword,
    authenticate,
    signInS3,
}
