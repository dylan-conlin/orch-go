import { DropdownMenu as DropdownMenuPrimitive } from "bits-ui";
import Content from "./dropdown-menu-content.svelte";
import Item from "./dropdown-menu-item.svelte";
import GroupHeading from "./dropdown-menu-group-heading.svelte";
import Separator from "./dropdown-menu-separator.svelte";
import RadioGroup from "./dropdown-menu-radio-group.svelte";
import RadioItem from "./dropdown-menu-radio-item.svelte";

const Root = DropdownMenuPrimitive.Root;
const Trigger = DropdownMenuPrimitive.Trigger;
const Group = DropdownMenuPrimitive.Group;

export {
	Root,
	Trigger,
	Content,
	Item,
	GroupHeading,
	Separator,
	Group,
	RadioGroup,
	RadioItem,
	//
	Root as DropdownMenu,
	Trigger as DropdownMenuTrigger,
	Content as DropdownMenuContent,
	Item as DropdownMenuItem,
	GroupHeading as DropdownMenuGroupHeading,
	Separator as DropdownMenuSeparator,
	Group as DropdownMenuGroup,
	RadioGroup as DropdownMenuRadioGroup,
	RadioItem as DropdownMenuRadioItem,
	// Alias for backward compatibility
	GroupHeading as Label,
	GroupHeading as DropdownMenuLabel,
};
