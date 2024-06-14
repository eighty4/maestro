/// <reference types="astro/client" />

declare interface Navigator {
    readonly userAgentData?: NavigatorUserAgentData;
}

interface NavigatorUserAgentData {
    readonly platform?: string;
}
