/** 仅浏览器入口加载；把 xterm 挂到全局供工作台使用。 */
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";

globalThis.__HARBOR_XTERM__ = { Terminal, FitAddon };
