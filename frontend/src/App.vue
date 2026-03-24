<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { GetDevices, TakeScreenshot, GetUIHierarchy, CropAndSaveScreenshot, CheckUiAutomatorProcess, KillProcess, RebootDevice } from '../wailsjs/go/main/App';
import { useToast } from 'primevue/usetoast';
import UITreeNode from './components/UITreeNode.vue';

const toast = useToast();

const devices = ref<any[]>([]);
const selectedDevice = ref<any>(null);
const loading = ref(false);

const screenshotData = ref('');
const uiTree = ref<any>(null);

const selectedNodeId = ref<string>('');
const selectedNode = ref<any>(null);
const hoveredNode = ref<any>(null);
const expandedNodes = ref<Record<string, boolean>>({});

// Image and interaction state
const imgRef = ref<HTMLImageElement | null>(null);
const showCoordinates = ref(false);
const mouseCoordinates = ref({ x: 0, y: 0 });

let lastClickInfo = { x: -1, y: -1, index: 0, matches: [] as any[] };

// Crop mode state
const isCropMode = ref(false);
const cropStart = ref({ x: 0, y: 0 });
const cropEnd = ref({ x: 0, y: 0 });
const isCropping = ref(false);
const hasCropSelection = ref(false);

const toggleCropMode = () => {
  isCropMode.value = !isCropMode.value;
  if (isCropMode.value) {
    selectedNodeId.value = '';
    selectedNode.value = null;
    hoveredNode.value = null;
    hasCropSelection.value = false;
  } else {
    hasCropSelection.value = false;
  }
};

const getCropStyle = (): any => {
  if (!hasCropSelection.value && !isCropping.value) return { display: 'none' };
  if (!imgRef.value) return { display: 'none' };
  
  const natW = imgRef.value.naturalWidth;
  const natH = imgRef.value.naturalHeight;
  
  const left = Math.min(cropStart.value.x, cropEnd.value.x);
  const top = Math.min(cropStart.value.y, cropEnd.value.y);
  const width = Math.abs(cropEnd.value.x - cropStart.value.x);
  const height = Math.abs(cropEnd.value.y - cropStart.value.y);
  
  const leftPct = (left / natW) * 100;
  const topPct = (top / natH) * 100;
  const widthPct = (width / natW) * 100;
  const heightPct = (height / natH) * 100;

  return {
    left: `${leftPct}%`,
    top: `${topPct}%`,
    width: `${widthPct}%`,
    height: `${heightPct}%`,
    border: '2px solid #3b82f6',
    backgroundColor: 'rgba(59, 130, 246, 0.2)',
    position: 'absolute',
    pointerEvents: 'none',
    zIndex: 20,
    boxSizing: 'border-box'
  };
};

const saveCrop = async () => {
  if (!hasCropSelection.value || !screenshotData.value) return;
  
  const left = Math.min(cropStart.value.x, cropEnd.value.x);
  const top = Math.min(cropStart.value.y, cropEnd.value.y);
  const width = Math.abs(cropEnd.value.x - cropStart.value.x);
  const height = Math.abs(cropEnd.value.y - cropStart.value.y);
  
  if (width === 0 || height === 0) {
    toast.add({ severity: 'warn', summary: 'Warning', detail: 'Invalid crop area', life: 2000 });
    return;
  }
  
  try {
    const defaultName = `crop_${selectedDevice.value?.id || 'device'}_${Date.now()}.png`;
    await CropAndSaveScreenshot(screenshotData.value, Math.round(left), Math.round(top), Math.round(width), Math.round(height), defaultName);
    toast.add({ severity: 'success', summary: 'Success', detail: 'Screenshot saved', life: 2000 });
    isCropMode.value = false;
    hasCropSelection.value = false;
  } catch (error: any) {
    toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to save: ' + error, life: 3000 });
  }
};

const getDeviceIcon = (deviceId: string) => {
  if (deviceId.includes(':') || deviceId.match(/^\d+\.\d+\.\d+\.\d+/)) {
    return 'pi pi-wifi';
  }
  return 'pi pi-usb';
};

const loadDevices = async () => {
  try {
    const list = await GetDevices();
    devices.value = list || [];
    if (devices.value.length > 0 && !selectedDevice.value) {
      selectedDevice.value = devices.value[0];
    }
    toast.add({ severity: 'success', summary: 'Success', detail: 'Devices refreshed', life: 2000 });
  } catch (error: any) {
    toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to load devices: ' + error, life: 3000 });
  }
};

// Process conflict state
const showConflictDialog = ref(false);
const conflictProcessInfo = ref<any>(null);
const isKilling = ref(false);

const handleProcessConflict = async (devId: string) => {
  try {
    const processInfo = await CheckUiAutomatorProcess(devId);
    if (processInfo && processInfo.pid) {
      conflictProcessInfo.value = processInfo;
      showConflictDialog.value = true;
    } else {
      // If we couldn't find a specific process, it's unknown.
      conflictProcessInfo.value = { package: 'Unknown Application', pid: 'Unknown', isUnknown: true };
      showConflictDialog.value = true;
    }
  } catch (err) {
    toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to check process status.', life: 3000 });
  }
};

const killConflictAndRetry = async () => {
  if (!conflictProcessInfo.value || !selectedDevice.value) return;
  isKilling.value = true;
  
  try {
    if (conflictProcessInfo.value.isUnknown) {
      // Reboot device if unknown process
      await RebootDevice(selectedDevice.value.id);
      toast.add({ severity: 'info', summary: 'Rebooting', detail: 'Device is rebooting. Please wait...', life: 5000 });
      showConflictDialog.value = false;
      // Clear current states
      screenshotData.value = '';
      uiTree.value = null;
    } else {
      // Normal kill/force-stop
      await KillProcess(selectedDevice.value.id, conflictProcessInfo.value.pid, conflictProcessInfo.value.package);
      toast.add({ severity: 'success', summary: 'Success', detail: 'Process stopped. Retrying...', life: 2000 });
      showConflictDialog.value = false;
      // Wait a brief moment for the device to recover before retrying
      setTimeout(() => {
        refreshData();
      }, 1500);
    }
  } catch (error: any) {
    toast.add({ severity: 'error', summary: 'Error', detail: 'Operation failed: ' + error, life: 3000 });
  } finally {
    isKilling.value = false;
  }
};

const refreshData = async () => {
  if (!selectedDevice.value) {
    toast.add({ severity: 'warn', summary: 'Warning', detail: 'Please select a device first', life: 3000 });
    return;
  }
  loading.value = true;
  try {
    const devId = selectedDevice.value.id;
    const [img, tree] = await Promise.all([
      TakeScreenshot(devId),
      GetUIHierarchy(devId)
    ]);
    screenshotData.value = img;
    uiTree.value = tree;
    expandedNodes.value = {};
    if (tree) {
      expandedNodes.value[tree.id] = true;
    }
    toast.add({ severity: 'success', summary: 'Success', detail: 'Screen refreshed', life: 2000 });
  } catch (error: any) {
    console.error("Refresh failed:", error);
    // Check if error contains dump failure, indicating uiautomator conflict
    if (error.toString().includes("failed to dump UI hierarchy") || error.toString().includes("uiautomator")) {
      await handleProcessConflict(selectedDevice.value.id);
    } else {
      toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to capture: ' + error, life: 3000 });
    }
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  loadDevices();
});

// Interactions
const handleNodeSelect = (node: any) => {
  selectedNodeId.value = node.id;
  selectedNode.value = node;
};

const handleNodeHover = (node: any) => {
  hoveredNode.value = node;
};

const handleToggleExpanded = (nodeId: string, isExpanded: boolean) => {
  expandedNodes.value[nodeId] = isExpanded;
};

const parseBounds = (boundsStr: string) => {
  if (!boundsStr) return null;
  // format: [left,top][right,bottom]
  const match = boundsStr.match(/\[(\d+),(\d+)\]\[(\d+),(\d+)\]/);
  if (match) {
    return {
      left: parseInt(match[1]),
      top: parseInt(match[2]),
      right: parseInt(match[3]),
      bottom: parseInt(match[4]),
      width: parseInt(match[3]) - parseInt(match[1]),
      height: parseInt(match[4]) - parseInt(match[2])
    };
  }
  return null;
};

const getHighlightStyle = (node: any, isHover: boolean): any => {
  if (!node || !imgRef.value || !screenshotData.value) return { display: 'none' };
  const bounds = parseBounds(node.bounds);
  if (!bounds) return { display: 'none' };

  // Calculate scaling factor between real device resolution and image displayed resolution
  // Assuming natural width/height of the img tag matches device resolution
  const natW = imgRef.value.naturalWidth;
  const natH = imgRef.value.naturalHeight;
  if (!natW || !natH) return { display: 'none' };

  // However, we apply bounds on a relative container, we can use percentages
  const leftPct = (bounds.left / natW) * 100;
  const topPct = (bounds.top / natH) * 100;
  const widthPct = (bounds.width / natW) * 100;
  const heightPct = (bounds.height / natH) * 100;

  return {
    left: `${leftPct}%`,
    top: `${topPct}%`,
    width: `${widthPct}%`,
    height: `${heightPct}%`,
    border: isHover ? '2px dashed #fde047' : '2px solid #ef4444',
    backgroundColor: isHover ? 'rgba(253, 224, 71, 0.2)' : 'rgba(239, 68, 68, 0.15)',
    boxShadow: isHover ? 'none' : '0 0 0 1px #fff, inset 0 0 0 1px #fff',
    position: 'absolute',
    pointerEvents: 'none',
    zIndex: isHover ? 10 : 5,
    boxSizing: 'border-box'
  };
};

const findAllNodesByBounds = (tree: any, x: number, y: number): any[] => {
  if (!tree) return [];
  let matches: any[] = [];
  const bounds = parseBounds(tree.bounds);
  if (bounds && x >= bounds.left && x <= bounds.right && y >= bounds.top && y <= bounds.bottom) {
    matches.push(tree);
  }
  if (tree.children) {
    for (const child of tree.children) {
      matches.push(...findAllNodesByBounds(child, x, y));
    }
  }
  return matches;
};

const handleImgClick = (e: MouseEvent) => {
  if (!imgRef.value || !uiTree.value || isCropMode.value) return;

  if (selectedNode.value) {
    selectedNodeId.value = '';
    selectedNode.value = null;
    return;
  }

  const natW = imgRef.value.naturalWidth;
  const natH = imgRef.value.naturalHeight;
  
  // Use offsetX/offsetY relative to the actual displayed size
  // e.offsetX and e.offsetY are relative to the img element
  // Need to account for object-fit: contain scale differences
  const rect = imgRef.value.getBoundingClientRect();
  const scaleX = natW / rect.width;
  const scaleY = natH / rect.height;
  
  const x = e.offsetX * scaleX;
  const y = e.offsetY * scaleY;
  
  // If clicking near the same spot, cycle through matches
  const tolerance = 10; // pixel tolerance for same click
  if (Math.abs(lastClickInfo.x - e.offsetX) < tolerance && Math.abs(lastClickInfo.y - e.offsetY) < tolerance && lastClickInfo.matches.length > 0) {
    lastClickInfo.index = (lastClickInfo.index + 1) % lastClickInfo.matches.length;
  } else {
    let matches = findAllNodesByBounds(uiTree.value, x, y);
    // Sort by area ascending (smallest first)
    matches.sort((a, b) => {
      const bA = parseBounds(a.bounds);
      const bB = parseBounds(b.bounds);
      const areaA = bA ? bA.width * bA.height : 0;
      const areaB = bB ? bB.width * bB.height : 0;
      return areaA - areaB;
    });
    lastClickInfo = { x: e.offsetX, y: e.offsetY, index: 0, matches };
  }

  if (lastClickInfo.matches.length > 0) {
    const node = lastClickInfo.matches[lastClickInfo.index];
    handleNodeSelect(node);
    expandToNode(uiTree.value, node.id);
  } else {
    selectedNodeId.value = '';
    selectedNode.value = null;
  }
};

const handleImgMouseMove = (e: MouseEvent) => {
  if (!imgRef.value || !uiTree.value) return;
  const natW = imgRef.value.naturalWidth;
  const natH = imgRef.value.naturalHeight;
  
  const rect = imgRef.value.getBoundingClientRect();
  const scaleX = natW / rect.width;
  const scaleY = natH / rect.height;
  
  const x = e.offsetX * scaleX;
  const y = e.offsetY * scaleY;
  
  mouseCoordinates.value = { x: Math.round(x), y: Math.round(y) };
  showCoordinates.value = true;

  if (isCropMode.value) {
    if (isCropping.value) {
      cropEnd.value = { x, y };
      hasCropSelection.value = true;
    }
    return;
  }

  if (selectedNode.value) {
    hoveredNode.value = null;
    return;
  }

  let matches = findAllNodesByBounds(uiTree.value, x, y);
  if (matches.length > 0) {
    matches.sort((a, b) => {
      const bA = parseBounds(a.bounds);
      const bB = parseBounds(b.bounds);
      const areaA = bA ? bA.width * bA.height : 0;
      const areaB = bB ? bB.width * bB.height : 0;
      return areaA - areaB;
    });
    hoveredNode.value = matches[0]; // Highlight smallest node
  } else {
    hoveredNode.value = null;
  }
};

const handleImgMouseLeave = () => {
  showCoordinates.value = false;
  hoveredNode.value = null;
  if (isCropping.value) {
    isCropping.value = false;
  }
};

const expandToNode = (root: any, targetId: string): boolean => {
  if (root.id === targetId) return true;
  if (root.children) {
    for (const child of root.children) {
      if (expandToNode(child, targetId)) {
        expandedNodes.value[root.id] = true;
        return true;
      }
    }
  }
  return false;
};

// Zoom and Pan handlers
const handleWheel = (e: WheelEvent) => {
  e.preventDefault();
  /* no transform scale logic */
};

const handleMouseDown = (e: MouseEvent) => {
  if (!isCropMode.value || !imgRef.value) return;
  e.preventDefault();
  
  const natW = imgRef.value.naturalWidth;
  const natH = imgRef.value.naturalHeight;
  const rect = imgRef.value.getBoundingClientRect();
  const scaleX = natW / rect.width;
  const scaleY = natH / rect.height;
  
  const x = e.offsetX * scaleX;
  const y = e.offsetY * scaleY;
  
  cropStart.value = { x, y };
  cropEnd.value = { x, y };
  isCropping.value = true;
  hasCropSelection.value = false;
};

const handleMouseUp = () => {
  if (isCropMode.value && isCropping.value) {
    isCropping.value = false;
  }
};

const copyToClipboard = (text: string) => {
  if (!text) return;
  navigator.clipboard.writeText(text).then(() => {
    toast.add({ severity: 'success', summary: 'Copied', detail: 'Copied to clipboard', life: 2000 });
  }).catch(err => {
    toast.add({ severity: 'error', summary: 'Error', detail: 'Failed to copy', life: 2000 });
  });
};
</script>

<template>
  <div class="app-container">
    <Toast position="top-right" class="small-toast" />
    <div class="header">
      <div class="title-container">
        <div class="title">T-Viewer</div>
        <div class="version">v1.0.0</div>
      </div>
      <div class="toolbar">
        <Dropdown 
          v-model="selectedDevice" 
          :options="devices" 
          optionLabel="id" 
          placeholder="Select a Device" 
          class="device-select p-dropdown-sm"
          style="height: 36px; display: flex; align-items: center;"
        >
          <template #value="slotProps">
            <div v-if="slotProps.value" class="flex align-items-center">
              <i :class="getDeviceIcon(slotProps.value.id)" class="mr-2"></i>
              <div>{{ slotProps.value.id }}</div>
            </div>
            <span v-else>{{ slotProps.placeholder }}</span>
          </template>
          <template #option="slotProps">
            <div class="flex align-items-center">
              <i :class="getDeviceIcon(slotProps.option.id)" class="mr-2"></i>
              <div>{{ slotProps.option.id }}</div>
            </div>
          </template>
        </Dropdown>
        <Button icon="pi pi-refresh" @click="loadDevices" class="p-button-text p-button-sm" title="Refresh Devices" />
        <Button label="Refresh Screen" icon="pi pi-camera" @click="refreshData" :loading="loading" class="p-button-primary p-button-sm ml-2" />
        <div class="crop-tools ml-2" v-if="screenshotData">
          <Button 
            :label="isCropMode ? 'Cancel Crop' : 'Crop'" 
            :icon="isCropMode ? 'pi pi-times' : 'pi pi-clone'" 
            @click="toggleCropMode" 
            :class="isCropMode ? 'p-button-secondary p-button-sm' : 'p-button-outlined p-button-secondary p-button-sm'" 
          />
          <Button 
            v-if="isCropMode && hasCropSelection"
            label="Save Crop" 
            icon="pi pi-save" 
            @click="saveCrop" 
            class="p-button-success p-button-sm ml-2" 
          />
        </div>
      </div>
    </div>

    <div class="main-content">
      <!-- Left: Screenshot -->
      <div class="left-panel">
        <div class="panel-header" style="display: flex; justify-content: space-between;">
          <span>Screenshot</span>
          <span v-if="showCoordinates" style="font-family: monospace; color: #6b7280; font-size: 0.8rem;">
            [ {{ mouseCoordinates.x }}, {{ mouseCoordinates.y }} ]
          </span>
        </div>
        <div class="screenshot-wrapper"
             @mousedown="handleMouseDown"
             @mouseup="handleMouseUp"
             @mouseleave="handleImgMouseLeave">
          <div v-if="screenshotData" class="transform-container">
            <div class="image-container" :class="{ 'crop-cursor': isCropMode }">
              <img :src="screenshotData" 
                   ref="imgRef" 
                   alt="Screenshot" 
                   draggable="false" 
                   @click="handleImgClick"
                   @mousemove="handleImgMouseMove"
              />
              <!-- Selected Node Highlight -->
              <div v-if="selectedNodeId && !isCropMode" :style="getHighlightStyle(uiTree ? findNodeById(uiTree, selectedNodeId) : null, false)"></div>
              <!-- Hovered Node Highlight -->
              <div v-if="hoveredNode && !selectedNode && !isCropMode" :style="getHighlightStyle(hoveredNode, true)"></div>
              <!-- Crop Selection Highlight -->
              <div v-if="isCropMode && (hasCropSelection || isCropping)" :style="getCropStyle()"></div>
            </div>
          </div>
          <div v-else class="empty-state">
            No screenshot available. Click "Refresh Screen".
          </div>
        </div>
      </div>

      <!-- Right: UI Tree & Details -->
      <div class="right-panel">
        <div class="panel-header">UI Hierarchy</div>
        <div class="tree-wrapper" style="flex: 1; overflow-y: auto; padding: 10px;">
          <UITreeNode 
            v-if="uiTree"
            :node="uiTree"
            :level="0"
            :selected-id="selectedNodeId"
            :expanded-nodes="expandedNodes"
            @select="handleNodeSelect"
            @hover="handleNodeHover"
            @toggle-expanded="handleToggleExpanded"
          />
          <div v-else class="empty-state">
            No UI hierarchy available.
          </div>
        </div>
        
        <!-- Details Panel -->
        <div class="details-panel" v-if="selectedNode">
          <div class="panel-header" style="border-top: 1px solid #e5e7eb; display: flex; justify-content: space-between;">
            <span>Node Details</span>
            <Button icon="pi pi-times" class="p-button-rounded p-button-text p-button-sm" @click="selectedNodeId = ''; selectedNode = null" style="width: 1.5rem; height: 1.5rem; padding: 0;" />
          </div>
          <div class="details-content">
            <div class="detail-row" v-if="selectedNode.id">
              <span class="detail-label">Index</span>
              <span class="detail-value">{{ selectedNode.index }}</span>
              <i class="pi pi-copy copy-icon" @click="copyToClipboard(selectedNode.index)" title="Copy"></i>
            </div>
            <div class="detail-row" v-if="selectedNode.class">
              <span class="detail-label">Class</span>
              <span class="detail-value">{{ selectedNode.class }}</span>
              <i class="pi pi-copy copy-icon" @click="copyToClipboard(selectedNode.class)" title="Copy"></i>
            </div>
            <div class="detail-row" v-if="selectedNode.text">
              <span class="detail-label">Text</span>
              <span class="detail-value">{{ selectedNode.text }}</span>
              <i class="pi pi-copy copy-icon" @click="copyToClipboard(selectedNode.text)" title="Copy"></i>
            </div>
            <div class="detail-row" v-if="selectedNode.contentDesc">
              <span class="detail-label">Content-desc</span>
              <span class="detail-value">{{ selectedNode.contentDesc }}</span>
              <i class="pi pi-copy copy-icon" @click="copyToClipboard(selectedNode.contentDesc)" title="Copy"></i>
            </div>
            <div class="detail-row" v-if="selectedNode.resourceId">
              <span class="detail-label">Resource-id</span>
              <span class="detail-value">{{ selectedNode.resourceId }}</span>
              <i class="pi pi-copy copy-icon" @click="copyToClipboard(selectedNode.resourceId)" title="Copy"></i>
            </div>
            <div class="detail-row" v-if="selectedNode.package">
              <span class="detail-label">Package</span>
              <span class="detail-value">{{ selectedNode.package }}</span>
              <i class="pi pi-copy copy-icon" @click="copyToClipboard(selectedNode.package)" title="Copy"></i>
            </div>
            <div class="detail-row" v-if="selectedNode.bounds">
              <span class="detail-label">Bounds</span>
              <span class="detail-value">{{ selectedNode.bounds }}</span>
              <i class="pi pi-copy copy-icon" @click="copyToClipboard(selectedNode.bounds)" title="Copy"></i>
            </div>
            <div class="detail-row flags-row">
              <span class="flag" :class="{active: selectedNode.clickable === 'true'}">Clickable</span>
              <span class="flag" :class="{active: selectedNode.enabled === 'true'}">Enabled</span>
              <span class="flag" :class="{active: selectedNode.focusable === 'true'}">Focusable</span>
              <span class="flag" :class="{active: selectedNode.scrollable === 'true'}">Scrollable</span>
              <span class="flag" :class="{active: selectedNode.checked === 'true'}">Checked</span>
              <span class="flag" :class="{active: selectedNode.selected === 'true'}">Selected</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Process Conflict Dialog -->
    <Dialog v-model:visible="showConflictDialog" modal header="Process Conflict Detected" :style="{ width: '450px' }">
      <div class="conflict-dialog-content">
        <i class="pi pi-exclamation-triangle text-red-500 mr-3" style="font-size: 2rem"></i>
        <div>
          <p class="m-0 mb-2"><strong>uiautomator</strong> is currently occupied by another process.</p>
          <div class="process-details" v-if="conflictProcessInfo">
            <div><span class="label">Package:</span> {{ conflictProcessInfo.package }}</div>
            <div v-if="!conflictProcessInfo.isUnknown"><span class="label">PID:</span> {{ conflictProcessInfo.pid }}</div>
          </div>
          <p class="m-0 mt-3 text-gray-600" v-if="conflictProcessInfo?.isUnknown">
            The occupying process could not be identified. Do you want to reboot the device to resolve this?
          </p>
          <p class="m-0 mt-3 text-gray-600" v-else>
            Do you want to force stop this process and retry?
          </p>
        </div>
      </div>
      <template #footer>
        <Button label="Cancel" icon="pi pi-times" @click="showConflictDialog = false" class="p-button-text" />
        <Button 
          :label="conflictProcessInfo?.isUnknown ? 'Reboot Device' : 'Force Stop & Retry'" 
          :icon="conflictProcessInfo?.isUnknown ? 'pi pi-power-off' : 'pi pi-check'" 
          @click="killConflictAndRetry" 
          class="p-button-danger" 
          :loading="isKilling" 
          autofocus 
        />
      </template>
    </Dialog>
  </div>
</template>

<script lang="ts">
export function findNodeById(tree: any, id: string): any {
  if (!tree) return null;
  if (tree.id === id) return tree;
  if (tree.children) {
    for (const child of tree.children) {
      const found = findNodeById(child, id);
      if (found) return found;
    }
  }
  return null;
}
</script>

<style scoped>
@font-face {
  font-family: 'OPPO Sans';
  src: url('./assets/fonts/OPPOSans40.woff2') format('woff2');
  font-weight: 400;
  font-style: normal;
}

.app-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background-color: #f3f4f6;
  color: #1f2937;
  font-family: "OPPO Sans", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 20px;
  background-color: #ffffff;
  border-bottom: 1px solid #e5e7eb;
  box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
}

.title-container {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.title {
  font-size: 1.25rem;
  font-weight: 600;
  color: #111827;
}

.version {
  font-size: 0.75rem;
  color: #9ca3af;
  font-weight: 500;
  padding: 2px 6px;
  background-color: #f3f4f6;
  border-radius: 4px;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
}

.device-select {
  width: 250px;
}

.main-content {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.left-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  border-right: 1px solid #e5e7eb;
  background-color: #f9fafb;
}

.right-panel {
  width: 400px;
  display: flex;
  flex-direction: column;
  background-color: #ffffff;
}

.panel-header {
  padding: 10px 15px;
  font-weight: 600;
  font-size: 0.9rem;
  background-color: #f3f4f6;
  border-bottom: 1px solid #e5e7eb;
  color: #4b5563;
}

.screenshot-wrapper {
  flex: 1;
  overflow: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  background-color: #f3f4f6;
  padding: 20px;
}

.transform-container {
  max-width: 100%;
  max-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.image-container {
  position: relative;
  display: inline-block;
  max-width: 100%;
  max-height: 100%;
}

.image-container img {
  max-width: 100%;
  max-height: calc(100vh - 120px);
  object-fit: contain;
  display: block;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
}

.crop-cursor {
  cursor: crosshair;
}

.crop-tools {
  display: flex;
  align-items: center;
}

.conflict-dialog-content {
  display: flex;
  align-items: flex-start;
  padding: 1rem 0;
}

.process-details {
  background-color: #f3f4f6;
  padding: 0.75rem;
  border-radius: 6px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 0.85rem;
  margin-top: 0.5rem;
}

.process-details .label {
  color: #6b7280;
  display: inline-block;
  width: 70px;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #9ca3af;
  font-size: 0.9rem;
}

.details-panel {
  display: flex;
  flex-direction: column;
  background-color: #fafafa;
  max-height: 40%;
  overflow-y: auto;
}

.details-content {
  padding: 10px 15px;
  font-size: 0.85rem;
}

.detail-row {
  display: flex;
  align-items: flex-start;
  margin-bottom: 6px;
  word-break: break-all;
  position: relative;
}

.detail-row:hover .copy-icon {
  opacity: 1;
}

.copy-icon {
  opacity: 0;
  cursor: pointer;
  color: #9ca3af;
  margin-left: 8px;
  font-size: 0.8rem;
  transition: opacity 0.2s, color 0.2s;
  padding-top: 2px;
}

.copy-icon:hover {
  color: #3b82f6;
}

.detail-label {
  font-weight: 600;
  color: #6b7280;
  width: 90px;
  flex-shrink: 0;
  font-family: "OPPO Sans", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
}

.detail-value {
  color: #111827;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  flex: 1;
}

.flags-row {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 8px;
}

.flag {
  padding: 2px 6px;
  background-color: #e5e7eb;
  color: #9ca3af;
  border-radius: 4px;
  font-size: 0.75rem;
}

.flag.active {
  background-color: #1f2937;
  color: #ffffff;
}

/* Override PrimeVue Toast default sizes */
:deep(.small-toast) {
  width: 260px !important;
  opacity: 0.95;
  top: 15px !important;
  right: 15px !important;
}

:deep(.small-toast .p-toast-message-content) {
  padding: 0.4rem 0.8rem !important;
  font-size: 0.8rem !important;
  align-items: center;
}

:deep(.small-toast .p-toast-message-icon) {
  font-size: 0.9rem !important;
  margin-right: 0.4rem !important;
}

:deep(.small-toast .p-toast-summary) {
  font-size: 0.85rem !important;
  margin-bottom: 0 !important;
}

:deep(.small-toast .p-toast-detail) {
  font-size: 0.8rem !important;
  margin-top: 0.1rem !important;
}

:deep(.small-toast .p-toast-icon-close) {
  width: 1.2rem !important;
  height: 1.2rem !important;
  margin-left: 0.5rem !important;
}
</style>
