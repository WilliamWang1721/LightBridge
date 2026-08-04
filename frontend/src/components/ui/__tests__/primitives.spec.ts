import { describe, expect, it } from 'vitest'
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Avatar } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { DataTableShell } from '@/components/ui/data-table'
import { Dialog, DialogContent, DialogFooter, DialogHeader } from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { PageHeader, PageSection, PageShell, PageState, PageToolbar } from '@/components/ui/page'
import { Pagination } from '@/components/ui/pagination'
import { Popover, PopoverContent } from '@/components/ui/popover'
import { Progress } from '@/components/ui/progress'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Select, SelectItem } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { Sheet, SheetContent } from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { Tooltip } from '@/components/ui/tooltip'

const exportsUnderTest = {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
  Alert,
  AlertDescription,
  AlertTitle,
  Avatar,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
  Checkbox,
  DataTableShell,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  Input,
  Label,
  PageHeader,
  PageSection,
  PageShell,
  PageState,
  PageToolbar,
  Pagination,
  Popover,
  PopoverContent,
  Progress,
  RadioGroup,
  RadioGroupItem,
  Select,
  SelectItem,
  Separator,
  Sheet,
  SheetContent,
  Skeleton,
  Switch,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  Textarea,
  Tooltip,
}

describe('Luma primitive library', () => {
  it('exports every supported primitive entrypoint', () => {
    for (const [name, component] of Object.entries(exportsUnderTest)) {
      expect(component, name).toBeTruthy()
    }
  })
})
