<script lang="ts">
  import logo from "../assets/mut-logo.png";
  export let subtitle = false;
</script>

<div class="brand">
  <img src={logo} alt="" />
  <div>
    mut{#if subtitle}<small>ЛИЧНЫЙ АРХИВ</small>{/if}
  </div>
</div>
