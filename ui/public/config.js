let configData = {
    BASE_URL_DEV: '/',
    BASE_URL_PROD: '/',
    XTermFontDefaultSize : 16,
    XTermFontMinSize : 12,
    XTermFontFamily : 'courier-new, courier, monospace'
}

const getConfigItem = (item_name) => {
    return configData[item_name];
}