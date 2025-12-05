package com.myfamily.agent;

import android.Manifest;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.net.http.SslError;
import android.os.Build;
import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.webkit.SslErrorHandler;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.Button;
import android.widget.CheckBox;
import android.widget.EditText;
import android.widget.TextView;
import android.widget.Toast;
import androidx.annotation.NonNull;
import androidx.appcompat.app.AlertDialog;
import androidx.appcompat.app.AppCompatActivity;
import androidx.core.app.ActivityCompat;
import androidx.core.content.ContextCompat;
import androidx.recyclerview.widget.RecyclerView;
import androidx.viewpager2.widget.ViewPager2;
import mfagent.Mfagent;

public class MainActivity extends AppCompatActivity {
    private static final int PERMISSION_REQUEST_CODE = 1001;
    private static final int PAGE_SETTINGS = 0;
    private static final int PAGE_WEBVIEW = 1;

    private ViewPager2 viewPager;
    private EditText deviceIdInput;
    private EditText deviceNameInput;
    private EditText endpointInput;
    private EditText usernameInput;
    private EditText passwordInput;
    private CheckBox skipTlsCheckbox;
    private TextView statusText;
    private Button startButton;
    private Button stopButton;
    private WebView webView;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);

        viewPager = findViewById(R.id.view_pager);
        viewPager.setAdapter(new PagerAdapter());

        // Load website when swiping to webview page
        viewPager.registerOnPageChangeCallback(new ViewPager2.OnPageChangeCallback() {
            @Override
            public void onPageSelected(int position) {
                if (position == PAGE_WEBVIEW && webView != null) {
                    loadWebsite();
                }
            }
        });

        // Initialize config
        Mfagent.setConfigDir(getFilesDir().getAbsolutePath());
        try {
            Mfagent.loadConfig();
        } catch (Exception e) {
            // Config file may not exist yet
        }
    }

    private void loadWebsite() {
        String endpoint = endpointInput != null ? endpointInput.getText().toString().trim() : "";
        if (endpoint.isEmpty()) {
            endpoint = Mfagent.getEndpoint();
        }
        if (!endpoint.isEmpty() && webView != null) {
            webView.loadUrl(endpoint);
        }
    }

    private void setupSettingsPage(View view) {
        deviceIdInput = view.findViewById(R.id.device_id_input);
        deviceNameInput = view.findViewById(R.id.device_name_input);
        endpointInput = view.findViewById(R.id.endpoint_input);
        usernameInput = view.findViewById(R.id.username_input);
        passwordInput = view.findViewById(R.id.password_input);
        skipTlsCheckbox = view.findViewById(R.id.skip_tls_checkbox);
        statusText = view.findViewById(R.id.status_text);
        startButton = view.findViewById(R.id.start_button);
        stopButton = view.findViewById(R.id.stop_button);

        // Pre-fill fields from saved config
        deviceIdInput.setText(Mfagent.getDeviceID());
        deviceNameInput.setText(Mfagent.getDeviceName());
        endpointInput.setText(Mfagent.getEndpoint());
        usernameInput.setText(Mfagent.getUser());
        skipTlsCheckbox.setChecked(Mfagent.getSkipTLSVerify());

        startButton.setOnClickListener(v -> startTracking());
        stopButton.setOnClickListener(v -> stopTracking());

        updateStatus();
    }

    private void setupWebViewPage(View view) {
        webView = view.findViewById(R.id.web_view);

        WebSettings webSettings = webView.getSettings();
        webSettings.setJavaScriptEnabled(true);
        webSettings.setDomStorageEnabled(true);
        webSettings.setLoadWithOverviewMode(true);
        webSettings.setUseWideViewPort(true);
        webSettings.setBuiltInZoomControls(true);
        webSettings.setDisplayZoomControls(false);
        webSettings.setMixedContentMode(WebSettings.MIXED_CONTENT_ALWAYS_ALLOW);

        webView.setWebViewClient(new WebViewClient() {
            @Override
            public boolean shouldOverrideUrlLoading(WebView view, String url) {
                view.loadUrl(url);
                return true;
            }

            @Override
            public void onReceivedSslError(WebView view, SslErrorHandler handler, SslError error) {
                // If skip TLS is checked, proceed despite SSL errors
                if (skipTlsCheckbox != null && skipTlsCheckbox.isChecked()) {
                    handler.proceed();
                } else {
                    handler.proceed(); // Allow by default for WebView
                }
            }
        });
    }

    private void startTracking() {
        if (!checkPermissions()) {
            requestPermissions();
            return;
        }

        String deviceId = deviceIdInput.getText().toString().trim();
        String deviceName = deviceNameInput.getText().toString().trim();
        String endpoint = endpointInput.getText().toString().trim();
        String username = usernameInput.getText().toString().trim();
        String password = passwordInput.getText().toString();

        if (endpoint.isEmpty()) {
            Toast.makeText(this, "Endpoint is required", Toast.LENGTH_SHORT).show();
            return;
        }

        if (deviceId.isEmpty() && deviceName.isEmpty()) {
            Toast.makeText(this, "Device Name is required for new devices", Toast.LENGTH_SHORT).show();
            return;
        }

        if (username.isEmpty() || password.isEmpty()) {
            Toast.makeText(this, "Username and Password are required", Toast.LENGTH_SHORT).show();
            return;
        }

        if (!deviceId.isEmpty()) {
            Mfagent.setDeviceID(deviceId);
        }
        Mfagent.setDeviceName(deviceName);
        Mfagent.setEndpoint(endpoint);
        Mfagent.setCredentials(username, password);
        Mfagent.setSkipTLSVerify(skipTlsCheckbox.isChecked());

        startButton.setEnabled(false);
        statusText.setText("Status: Authenticating...");

        new Thread(() -> {
            try {
                Mfagent.authenticate();
                completeRegistration();
            } catch (Exception e) {
                String errorMsg = e.getMessage();
                if (errorMsg != null && errorMsg.contains("TFA_REQUIRED")) {
                    // TFA is required - show TFA input dialog
                    runOnUiThread(() -> showTfaDialog());
                } else {
                    runOnUiThread(() -> {
                        Toast.makeText(this, "Failed: " + e.getMessage(), Toast.LENGTH_LONG).show();
                        statusText.setText("Status: Failed");
                        startButton.setEnabled(true);
                    });
                }
            }
        }).start();
    }

    private void showTfaDialog() {
        statusText.setText("Status: TFA Required");

        final EditText tfaInput = new EditText(this);
        tfaInput.setHint("Enter 6-digit code");
        tfaInput.setInputType(android.text.InputType.TYPE_CLASS_NUMBER);
        tfaInput.setMaxLines(1);

        new AlertDialog.Builder(this)
                .setTitle("Two-Factor Authentication")
                .setMessage("Enter the 6-digit code from your authenticator app")
                .setView(tfaInput)
                .setCancelable(false)
                .setPositiveButton("Verify", (dialog, which) -> {
                    String code = tfaInput.getText().toString().trim();
                    if (code.length() != 6) {
                        Toast.makeText(this, "Please enter a 6-digit code", Toast.LENGTH_SHORT).show();
                        startButton.setEnabled(true);
                        statusText.setText("Status: Stopped");
                        return;
                    }
                    statusText.setText("Status: Verifying TFA...");
                    verifyTfaCode(code);
                })
                .setNegativeButton("Cancel", (dialog, which) -> {
                    startButton.setEnabled(true);
                    statusText.setText("Status: Stopped");
                    Mfagent.clearTfaState();
                })
                .show();
    }

    private void verifyTfaCode(String code) {
        new Thread(() -> {
            try {
                Mfagent.verifyTfa(code);
                completeRegistration();
            } catch (Exception e) {
                runOnUiThread(() -> {
                    Toast.makeText(this, "TFA verification failed: " + e.getMessage(), Toast.LENGTH_LONG).show();
                    statusText.setText("Status: Failed");
                    startButton.setEnabled(true);
                    Mfagent.clearTfaState();
                });
            }
        }).start();
    }

    private void completeRegistration() {
        try {
            runOnUiThread(() -> statusText.setText("Status: Registering device..."));
            Mfagent.registerDevice();

            try {
                Mfagent.saveConfig();
            } catch (Exception e) {
                // Log but don't fail
            }

            runOnUiThread(() -> {
                deviceIdInput.setText(Mfagent.getDeviceID());

                Intent serviceIntent = new Intent(this, LocationService.class);
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                    startForegroundService(serviceIntent);
                } else {
                    startService(serviceIntent);
                }
                Toast.makeText(this, "Device registered - tracking started", Toast.LENGTH_SHORT).show();
                updateStatus();
            });
        } catch (Exception e) {
            runOnUiThread(() -> {
                Toast.makeText(this, "Registration failed: " + e.getMessage(), Toast.LENGTH_LONG).show();
                statusText.setText("Status: Failed");
                startButton.setEnabled(true);
            });
        }
    }

    private void stopTracking() {
        Intent serviceIntent = new Intent(this, LocationService.class);
        stopService(serviceIntent);
        Toast.makeText(this, "Location tracking stopped", Toast.LENGTH_SHORT).show();
        updateStatus();
    }

    private void updateStatus() {
        if (statusText == null || startButton == null || stopButton == null) return;

        if (LocationService.isRunning) {
            statusText.setText("Status: Running");
            startButton.setEnabled(false);
            stopButton.setEnabled(true);
        } else {
            statusText.setText("Status: Stopped");
            startButton.setEnabled(true);
            stopButton.setEnabled(false);
        }
    }

    private boolean checkPermissions() {
        boolean fineLocation = ContextCompat.checkSelfPermission(this,
                Manifest.permission.ACCESS_FINE_LOCATION) == PackageManager.PERMISSION_GRANTED;
        boolean coarseLocation = ContextCompat.checkSelfPermission(this,
                Manifest.permission.ACCESS_COARSE_LOCATION) == PackageManager.PERMISSION_GRANTED;
        return fineLocation && coarseLocation;
    }

    private void requestPermissions() {
        String[] permissions;
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            permissions = new String[]{
                    Manifest.permission.ACCESS_FINE_LOCATION,
                    Manifest.permission.ACCESS_COARSE_LOCATION,
                    Manifest.permission.ACCESS_BACKGROUND_LOCATION
            };
        } else {
            permissions = new String[]{
                    Manifest.permission.ACCESS_FINE_LOCATION,
                    Manifest.permission.ACCESS_COARSE_LOCATION
            };
        }
        ActivityCompat.requestPermissions(this, permissions, PERMISSION_REQUEST_CODE);
    }

    @Override
    public void onRequestPermissionsResult(int requestCode, @NonNull String[] permissions,
                                           @NonNull int[] grantResults) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults);
        if (requestCode == PERMISSION_REQUEST_CODE) {
            if (grantResults.length > 0 && grantResults[0] == PackageManager.PERMISSION_GRANTED) {
                startTracking();
            } else {
                Toast.makeText(this, "Location permission required", Toast.LENGTH_LONG).show();
            }
        }
    }

    @Override
    protected void onResume() {
        super.onResume();
        updateStatus();
    }

    @Override
    public void onBackPressed() {
        if (viewPager.getCurrentItem() == PAGE_WEBVIEW) {
            if (webView != null && webView.canGoBack()) {
                webView.goBack();
            } else {
                viewPager.setCurrentItem(PAGE_SETTINGS);
            }
        } else {
            super.onBackPressed();
        }
    }

    // ViewPager adapter
    private class PagerAdapter extends RecyclerView.Adapter<PagerAdapter.PageViewHolder> {
        @NonNull
        @Override
        public PageViewHolder onCreateViewHolder(@NonNull ViewGroup parent, int viewType) {
            LayoutInflater inflater = LayoutInflater.from(parent.getContext());
            View view;
            if (viewType == PAGE_SETTINGS) {
                view = inflater.inflate(R.layout.page_settings, parent, false);
            } else {
                view = inflater.inflate(R.layout.page_webview, parent, false);
            }
            return new PageViewHolder(view, viewType);
        }

        @Override
        public void onBindViewHolder(@NonNull PageViewHolder holder, int position) {
            if (position == PAGE_SETTINGS) {
                setupSettingsPage(holder.itemView);
            } else {
                setupWebViewPage(holder.itemView);
            }
        }

        @Override
        public int getItemCount() {
            return 2;
        }

        @Override
        public int getItemViewType(int position) {
            return position;
        }

        class PageViewHolder extends RecyclerView.ViewHolder {
            PageViewHolder(View itemView, int viewType) {
                super(itemView);
            }
        }
    }
}
