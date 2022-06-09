import type { Ref, ExtractPropTypes } from 'vue';
export declare const useModelToggleProps: {
    modelValue: import("element-plus/es/utils/props").BuildPropReturn<import("element-plus/es/utils/props").PropWrapper<boolean | null>, null, unknown, unknown, unknown>;
    'onUpdate:modelValue': FunctionConstructor;
};
export declare type UseModelToggleProps = ExtractPropTypes<typeof useModelToggleProps>;
export declare const useModelToggleEmits: string[];
export declare type ModelToggleParams = {
    indicator: Ref<boolean>;
    shouldHideWhenRouteChanges?: Ref<boolean>;
    shouldProceed?: () => boolean;
    onShow?: () => void;
    onHide?: () => void;
};
export declare const useModelToggle: ({ indicator, shouldHideWhenRouteChanges, shouldProceed, onShow, onHide, }: ModelToggleParams) => {
    hide: () => void;
    show: () => void;
    toggle: () => void;
};
