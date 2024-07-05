package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/flopp/go-findfont"
	"gopkg.in/yaml.v3"
	"image/color"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var logFile *os.File
var translations map[string]map[string]string
var currentLang string
var (
	defaultKeyStore         = "keystore.jks"
	defaultKeyAlias         = "apicker"
	defaultKeyStorePassword = "ge63bc7btn"
	defaultKeyPassword      = "kwzpjg4s5c"
	defaultDName            = "CN=apicker.com, OU=RD, O=., L=., S=., C=US"
)

func main() {
	// Open log file
	var err error
	logFile, err = os.OpenFile("logs/apicker.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Error opening log file:", err)
		return
	}
	defer logFile.Close()
	log.SetFlags(log.LstdFlags | log.Llongfile)
	// Set log output to file
	log.SetOutput(logFile)

	// Load user config
	err = loadConfig()
	if err != nil {
		log.Println("Error loading config:", err)
	}

	// Determine language
	currentLang = config.Language
	if currentLang == "" {
		currentLang = getSystemLanguage()
	}
	fmt.Println(currentLang)
	// Load translations
	translations, err = loadLanguageFiles()
	if err != nil {
		log.Println("Error loading language files:", err)
		translations, _ = loadLanguageFiles()
	}

	// if isTTY() {
	// 	runCLI()
	// } else {
	runGUI()
	// }
}

func runCLI() {
	apkFile := flag.String("apk", "path/to/your.apk", translations[currentLang]["apkFilePath"])
	domain := flag.String("domain", "", translations[currentLang]["domain"])
	keystore := flag.String("keystore", defaultKeyStore, translations[currentLang]["keystorePath"])
	keystorePassword := flag.String("keystorePassword", defaultKeyStorePassword, translations[currentLang]["keystorePassword"])
	keyAlias := flag.String("keyAlias", defaultKeyAlias, translations[currentLang]["keyAlias"])
	keyPassword := flag.String("keyPassword", defaultKeyPassword, translations[currentLang]["keyPassword"])
	dname := flag.String("dname", defaultDName, translations[currentLang]["dname"])

	flag.Parse()

	missingDeps := checkDependencies()
	if len(missingDeps) > 0 {
		log.Println(translations[currentLang]["missingDependencies"])
		for _, dep := range missingDeps {
			log.Println("missing dep", dep)
		}
		return
	}

	err := modifyAPK(*apkFile, *domain, *keystore, *keystorePassword, *keyAlias, *keyPassword, *dname)
	if err != nil {
		log.Println("Error modifying APK:", err)
	}
}

func runGUI() {
	// fontPaths := findfont.List()
	// for _, path := range fontPaths {
	// 	fmt.Println(path)
	// 	// if strings.Contains(path, "FiraCode") {
	// 	// 	os.Setenv("FYNE_FONT", path)
	// 	// 	break
	// 	// }
	// }
	myApp := app.New()
	myApp.Settings().SetTheme(&myTheme{"AAA"})
	fmt.Println(myApp.Settings().Theme())
	// myApp.Settings().SetTheme(theme.LightTheme())
	myWindow := myApp.NewWindow("汉字显示效果")
	myWindow.CenterOnScreen()
	myWindow.Resize(fyne.NewSize(800, 600))
	myWindow.SetTitle(translations[currentLang]["apkFilePath"])
	// 使用暗色主题
	myApp.Settings().SetTheme(theme.DarkTheme())
	fmt.Println(currentLang, translations[currentLang]["apkFilePath"])
	// 文件路径选择器
	apkPathLabel := widget.NewLabel(translations[currentLang]["apkFilePath"])
	apkPathEntry := widget.NewEntry()
	apkPathEntry.SetPlaceHolder(translations[currentLang]["selectAPKFile"])
	apkPathButton := widget.NewButton(translations[currentLang]["browse"], func() {
		dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				apkPathEntry.SetText(reader.URI().Path())
			}
		}, myWindow).Show()
	})

	// 其他输入框
	domainEntry := widget.NewEntry()
	fmt.Println(translations[currentLang]["domain"])
	domainEntry.SetPlaceHolder(translations[currentLang]["domain"])

	keystoreEntry := widget.NewEntry()
	keystoreEntry.SetPlaceHolder(translations[currentLang]["keystorePath"])
	keystoreEntry.SetText(defaultKeyStore)

	keystorePasswordEntry := widget.NewPasswordEntry()
	keystorePasswordEntry.SetPlaceHolder(translations[currentLang]["keystorePassword"])
	keystorePasswordEntry.SetText(defaultKeyStorePassword)

	keyAliasEntry := widget.NewEntry()
	keyAliasEntry.SetPlaceHolder(translations[currentLang]["keyAlias"])
	keyAliasEntry.SetText(defaultKeyAlias)

	keyPasswordEntry := widget.NewPasswordEntry()
	keyPasswordEntry.SetPlaceHolder(translations[currentLang]["keyPassword"])
	keyPasswordEntry.SetText(defaultKeyPassword)
	dnameEntry := widget.NewEntry()
	dnameEntry.SetPlaceHolder(translations[currentLang]["dname"])
	dnameEntry.SetText(defaultDName)

	// 日志区域
	logArea := widget.NewMultiLineEntry()
	logArea.SetPlaceHolder(translations[currentLang]["logOutput"])
	logArea.Disable() // 禁用用户输入，使其成为只读

	// 限制日志区域的文本不超过2000行
	const maxLines = 2000
	appendLog := func(text string) {
		logLines := strings.Split(logArea.Text, "\n")
		logLines = append(logLines, text)
		if len(logLines) > maxLines {
			logLines = logLines[len(logLines)-maxLines:]
		}
		logArea.SetText(strings.Join(logLines, "\n"))
	}

	// 按钮点击事件
	button := widget.NewButton(translations[currentLang]["modifyAPK"], func() {
		apkFile := apkPathEntry.Text
		domain := domainEntry.Text
		keystore := keystoreEntry.Text
		keystorePassword := keystorePasswordEntry.Text
		keyAlias := keyAliasEntry.Text
		keyPassword := keyPasswordEntry.Text
		dname := dnameEntry.Text
		missing := checkDependencies()
		if len(missing) > 0 {
			aertDialog := dialog.NewCustom(translations[currentLang]["about"], translations[currentLang]["close"], container.NewVBox(
				widget.NewLabel(translations[currentLang]["author"]),
				widget.NewRichTextFromMarkdown(`
### you are missing dependencies
`+strings.Join(missing, " ")),
			), myWindow)
			aertDialog.Show()
		}
		appendLog(translations[currentLang]["apkModificationStarted"])
		// 调用 modifyAPK 函数
		err := modifyAPK(apkFile, domain, keystore, keystorePassword, keyAlias, keyPassword, dname)
		if err != nil {
			appendLog(fmt.Sprintf(translations[currentLang]["error"], err))
		} else {
			appendLog(translations[currentLang]["apkModificationCompleted"])
		}
	})

	// 关于按钮
	aboutButton := widget.NewButton(translations[currentLang]["about"], func() {
		aboutDialog := dialog.NewCustom(translations[currentLang]["about"], translations[currentLang]["close"], container.NewVBox(
			widget.NewLabel(translations[currentLang]["author"]),
			widget.NewLabel(translations[currentLang]["donate"]),
			widget.NewLabel(translations[currentLang]["alipay"]),
			widget.NewLabel(translations[currentLang]["wechat"]),
		), myWindow)
		aboutDialog.Show()
	})
	var languageSelect *widget.Select
	// 语言选择下拉菜单
	languageSelect = widget.NewSelect([]string{"English", "简体中文", "繁體中文", "日本語", "한국어"}, func(selected string) {
		switch selected {
		case "English":
			currentLang = "en"
		case "简体中文":
			currentLang = "zh"
		case "繁體中文":
			currentLang = "zh-TW"
		case "日本語":
			currentLang = "ja"
		case "한국어":
			currentLang = "ko"
		}
		config.Language = currentLang
		saveConfig()
		updateUI(apkPathLabel, apkPathEntry, apkPathButton, domainEntry, keystoreEntry, keystorePasswordEntry, keyAliasEntry, keyPasswordEntry, dnameEntry, logArea, button, aboutButton, languageSelect)
	})

	// 布局
	content := container.NewVBox(
		widget.NewLabel("师姐值大雾"),
		apkPathLabel,
		container.NewHBox(apkPathEntry, apkPathButton),
		widget.NewLabel(translations[currentLang]["domain"]),
		domainEntry,
		widget.NewLabel(translations[currentLang]["keystorePath"]),
		keystoreEntry,
		widget.NewLabel(translations[currentLang]["keystorePassword"]),
		keystorePasswordEntry,
		widget.NewLabel(translations[currentLang]["keyAlias"]),
		keyAliasEntry,
		widget.NewLabel(translations[currentLang]["keyPassword"]),
		keyPasswordEntry,
		widget.NewLabel(translations[currentLang]["dname"]),
		dnameEntry,
		widget.NewLabel(translations[currentLang]["logOutput"]),
		logArea,
		container.NewHBox(button, aboutButton, languageSelect),
	)

	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}

func updateUI(apkPathLabel *widget.Label, apkPathEntry *widget.Entry, apkPathButton *widget.Button, domainEntry, keystoreEntry, keystorePasswordEntry, keyAliasEntry, keyPasswordEntry, dnameEntry, logArea *widget.Entry, button, aboutButton *widget.Button, languageSelect *widget.Select) {
	apkPathLabel.SetText(translations[currentLang]["apkFilePath"])
	apkPathEntry.SetPlaceHolder(translations[currentLang]["selectAPKFile"])
	apkPathButton.SetText(translations[currentLang]["browse"])
	domainEntry.SetPlaceHolder(translations[currentLang]["domain"])
	keystoreEntry.SetPlaceHolder(translations[currentLang]["keystorePath"])
	keystorePasswordEntry.SetPlaceHolder(translations[currentLang]["keystorePassword"])
	keyAliasEntry.SetPlaceHolder(translations[currentLang]["keyAlias"])
	keyPasswordEntry.SetPlaceHolder(translations[currentLang]["keyPassword"])
	dnameEntry.SetPlaceHolder(translations[currentLang]["dname"])
	logArea.SetPlaceHolder(translations[currentLang]["logOutput"])
	button.SetText(translations[currentLang]["modifyAPK"])
	aboutButton.SetText(translations[currentLang]["about"])
	languageSelect.PlaceHolder = translations[currentLang]["selectLanguage"]
}

func modifyAPK(apkFile, domain, keystore, keystorePassword, keyAlias, keyPassword, dname string) error {
	outputDir := "output"
	os.RemoveAll(outputDir)

	// Step 1: Decode APK
	log.Println("Decoding APK...")
	decompileCmd := exec.Command("apktool", "d", apkFile, "-o", outputDir)
	fmt.Println(decompileCmd.Args)
	decompileCmd.Stdout = os.Stdout
	decompileCmd.Stderr = os.Stderr
	if err := decompileCmd.Run(); err != nil {
		log.Println("Error decoding APK:", err)
		return err
	}
	// return nil

	// Step 2: Modify AndroidManifest.xml
	manifestPath := outputDir + "/AndroidManifest.xml"
	log.Println("Modifying AndroidManifest.xml...")
	manifestContentBytes, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		log.Println("Error reading AndroidManifest.xml:", err)
		return err
	}

	// 解析 XML 文件
	var m manifest
	err = xml.NewDecoder(bytes.NewBuffer(manifestContentBytes)).Decode(&m)
	if err != nil {
		return fmt.Errorf("无法解析 AndroidManifest.xml: %v", err)
	}
	modifiedApk := fmt.Sprintf("%s_modified.apk", m.Package)

	// 删除解压后的文件夹

	oldAttrPattern := regexp.MustCompile(`android:networkSecurityConfig="@xml/[^"]+"`)

	// 新的 networkSecurityConfig 属性值
	newConfig := `android:networkSecurityConfig="@xml/network_security_config"`
	manifestContent := string(manifestContentBytes)
	// 查找并替换属性值
	if oldAttrPattern.MatchString(manifestContent) {
		manifestContent = oldAttrPattern.ReplaceAllString(manifestContent, newConfig)
	} else {
		// 如果属性不存在，则在 <application> 标签中添加
		reApp := regexp.MustCompile(`<application[^>]*>`)
		manifestContent = reApp.ReplaceAllString(manifestContent, `${0} `+newConfig)
	}

	if err := ioutil.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		log.Println("Error writing modified AndroidManifest.xml:", err)
		return err
	}

	// Step 3: Add network_security_config.xml
	log.Println("Adding network_security_config.xml...")
	var networkSecurityConfig string
	if domain != "" {
		networkSecurityConfig = fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<network-security-config>
    <domain-config cleartextTrafficPermitted="true">
        <domain includeSubdomains="true">%s</domain>
        <trust-anchors>
            <certificates src="system" />
            <certificates src="user" />
        </trust-anchors>
    </domain-config>
</network-security-config>`, domain)
	} else {
		networkSecurityConfig = `<?xml version="1.0" encoding="utf-8"?>
<network-security-config>
    <base-config cleartextTrafficPermitted="true">
        <trust-anchors>
            <certificates src="system" />
            <certificates src="user" />
        </trust-anchors>
    </base-config>
</network-security-config>`
	}

	resDir := outputDir + "/res/xml"
	if err := os.MkdirAll(resDir, 0755); err != nil {
		log.Println("Error creating res/xml directory:", err)
		return err
	}

	networkSecurityConfigPath := resDir + "/network_security_config.xml"
	if err := ioutil.WriteFile(networkSecurityConfigPath, []byte(networkSecurityConfig), 0644); err != nil {
		log.Println("Error writing network_security_config.xml:", err)
		return err
	}

	// Step 4: Rebuild APK
	log.Println("Rebuilding APK...")
	cmd := exec.Command("apktool", "b", outputDir, "-o", modifiedApk)
	fmt.Println(cmd.Args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Println("Error rebuilding APK:", err)
		return err
	}

	// Step 5: Check if keystore exists, if not generate a new one
	if _, err := os.Stat(keystore); os.IsNotExist(err) {
		log.Println("Keystore not found, generating a new one...")
		keytoolCmd := exec.Command("keytool", "-genkeypair", "-v", "-storetype", "JKS", "-keystore", keystore, "-storepass", keystorePassword, "-keypass", keyPassword, "-alias", keyAlias, "-keyalg", "RSA", "-keysize", "2048", "-validity", "10000", "-dname", dname)
		log.Println(keytoolCmd.Args)
		keytoolCmd.Stdout = os.Stdout
		keytoolCmd.Stderr = os.Stderr
		if err := keytoolCmd.Run(); err != nil {
			log.Println("Error generating keystore:", err)
			return err
		}
	}
	{
		checkKeyCmd := exec.Command("keytool", "-list", "-v", "-keystore", keystore, "-storepass", keystorePassword)
		log.Println(checkKeyCmd.Args)
		checkKeyCmd.Stdout = os.Stdout
		checkKeyCmd.Stderr = os.Stderr
		if err := checkKeyCmd.Run(); err != nil {
			log.Println("checkKeyCmd error:", err)
			return err
		}
	}
	signedModifedApk := "signed_" + modifiedApk
	// Step 6: Sign the APK
	log.Println("Signing APK...")
	signCmd := exec.Command("jarsigner", "-keystore", keystore, "-storepass", keystorePassword, "-keypass", keyPassword, "-signedjar", signedModifedApk, modifiedApk, keyAlias)
	log.Println(signCmd.Args)

	signCmd.Stdout = os.Stdout
	signCmd.Stderr = os.Stderr
	if err := signCmd.Run(); err != nil {
		log.Println("Error signing APK:", err)
		return err
	}

	log.Println("APK modified, rebuilt, and signed successfully:  " + signedModifedApk)

	device, err := getConnectedDevice()
	if err == nil && device != "" {
		log.Println("检测到设备:", device)
		err = uninstallAPK(device, m.Package)
		if err != nil {
			fmt.Println("Uninstall Error:", err)
		}

		// 安装新的 APK
		log.Println("安装新的 APK...")
		err = installAPK(device, signedModifedApk)
		if err != nil {
			log.Println("Error:", err)
			return err
		}
		log.Println("已经安装新的 APK...")
		// 启动应用
		fmt.Println("启动应用...")
		err = startApp(device, m.Package, m.Activity.Name)
		if err != nil {
			log.Println("启动应用 Error:", err)
			return err
		}
		fmt.Println("APK 安装并启动完成。")
	} else {
		log.Println("没有检测到设备，请手动安装:", signedModifedApk)
	}
	return nil
}

func checkDependencies() []string {
	dependencies := []string{"apktool", "keytool", "jarsigner"}
	missingDeps := []string{}

	for _, dep := range dependencies {
		if !isCommandAvailable(dep) {
			missingDeps = append(missingDeps, dep)
		}
	}

	return missingDeps
}

func isCommandAvailable(name string) bool {
	cmd := exec.Command("which", name)
	if runtime.GOOS == "windows" {
		cmd = exec.Command("where", name)
	}
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func isTTY() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func loadLanguageFiles() (map[string]map[string]string, error) {
	languages := []string{"en", "zh", "zh-TW", "ja", "ko"}
	translations := make(map[string]map[string]string)

	for _, lang := range languages {
		content, err := ioutil.ReadFile(fmt.Sprintf("%s.yaml", lang))
		if err != nil {
			fmt.Println("can not load lang file, use bundled one", lang)
			switch lang {
			case "zh":
				content = resourceZhYaml.Content()
			case "zh-TW":
				content = resourceZhTWYaml.Content()
			case "ja":
				content = resourceJaYaml.Content()
			case "ko":
				content = resourceKoYaml.Content()

			case "en":
				fallthrough
			default:

				content = resourceEnYaml.Content()
			}
			// return nil, err
		}

		var translation map[string]string
		if err := yaml.Unmarshal(content, &translation); err != nil {
			return nil, err
		}
		translations[lang] = translation
	}

	return translations, nil
}

func getSystemLanguage() string {
	lang := os.Getenv("LANG")
	if lang == "" {
		lang = os.Getenv("LC_ALL")
	}
	if lang == "" {
		lang = os.Getenv("LC_MESSAGES")
	}
	if lang == "" {
		lang = os.Getenv("LANGUAGE")
	}

	if lang == "" {
		return "en"
	}

	lang = strings.Split(lang, ".")[0]
	lang = strings.Split(lang, "_")[0]

	return lang
}

type Config struct {
	Language string `yaml:"language"`
}

func getConfigFilePath() string {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		os.Mkdir(configDir, 0755)
	}
	return filepath.Join(configDir, "apicker.yml")
}

var config Config

func loadConfig() (err error) {
	configFilePath := getConfigFilePath()

	if _, err = os.Stat(configFilePath); os.IsNotExist(err) {
		return
	}

	content, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return
	}

	if err = yaml.Unmarshal(content, &config); err != nil {
		return
	}

	return
}

func saveConfig() error {
	configFilePath := getConfigFilePath()
	content, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(configFilePath, content, 0644); err != nil {
		return err
	}

	return nil
}

type myTheme struct {
	Name string
}

var _ fyne.Theme = (*myTheme)(nil)

// return bundled font resource
func (*myTheme) Font(s fyne.TextStyle) fyne.Resource {
	fontPaht, err := findfont.Find("FiraCode")
	if err != nil {
		panic(err)
	}
	fari, err := fyne.LoadResourceFromPath(fontPaht)
	if err != nil {
		panic(err)
	}
	defaultFont := theme.DefaultTheme().Font(s)
	fmt.Println("font checking:::", defaultFont, s.Monospace, s.Bold, s.Italic)
	if s.Monospace {
		fmt.Println("getting Monospace")
		return defaultFont
	}
	if s.Bold {
		if s.Italic {
			return defaultFont
		}
		return fari
	}
	if s.Italic {
		return defaultFont
	}
	return fari

}

func (*myTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(n, v)
}

func (*myTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (*myTheme) Size(n fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(n)
}

type manifest struct {
	Package  string `xml:"package,attr"`
	Activity struct {
		Name string `xml:"name,attr"`
	} `xml:"application>activity"`
}

func getConnectedDevice() (string, error) {
	cmd := exec.Command("adb", "devices")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "device") && !strings.Contains(line, "List of devices attached") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				return parts[0], nil
			}
		}
	}
	return "", nil
}

func uninstallAPK(device, packageName string) error {
	cmd := exec.Command("adb", "-s", device, "uninstall", packageName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("uninstall error: %v, %s", err, stderr.String())
	}
	return nil
}

func installAPK(device, apkPath string) error {
	cmd := exec.Command("adb", "-s", device, "install", "-r", apkPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("install error: %v, %s", err, stderr.String())
	}
	return nil
}

func startApp(device, packageName, mainActivity string) error {
	cmd := exec.Command("adb", "-s", device, "shell", "am", "start", "-n", fmt.Sprintf("%s/%s", packageName, mainActivity))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("start app error: %v, %s", err, stderr.String())
	}
	return nil
}
